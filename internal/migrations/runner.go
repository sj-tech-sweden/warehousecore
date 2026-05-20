package migrations

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const blockCommentMarkerLen = 2

// Arbitrary application-specific advisory lock ID used to serialize startup migrations.
// Keeping this constant stable ensures all app instances contend on the same lock.
const migrationsAdvisoryLockKey int64 = 871234901
const migrationsAdvisoryLockTimeout = 2 * time.Minute
const migrationsAdvisoryLockRetryInterval = time.Second

func isForwardMigrationFile(name string) bool {
	if !strings.HasSuffix(name, ".sql") {
		return false
	}
	lower := strings.ToLower(name)
	if strings.Contains(lower, ".down.") || strings.HasSuffix(lower, ".down.sql") || strings.HasSuffix(lower, "_down.sql") {
		return false
	}
	return true
}

func parseSQLStatements(sqlText string) []string {
	lines := strings.Split(sqlText, "\n")
	cleaned := make([]string, 0, len(lines))
	inBlockComment := false

	for _, line := range lines {
		remaining := line

		if inBlockComment {
			if end := strings.Index(remaining, "*/"); end >= 0 {
				remaining = remaining[end+blockCommentMarkerLen:]
				inBlockComment = false
			} else {
				continue
			}
		}

		for {
			start := strings.Index(remaining, "/*")
			if start < 0 {
				break
			}
			end := strings.Index(remaining[start+blockCommentMarkerLen:], "*/")
			if end < 0 {
				remaining = remaining[:start]
				inBlockComment = true
				break
			}
			remaining = remaining[:start] + remaining[start+blockCommentMarkerLen+end+blockCommentMarkerLen:]
		}

		if idx := strings.Index(remaining, "--"); idx >= 0 {
			remaining = remaining[:idx]
		}

		cleaned = append(cleaned, remaining)
	}

	joined := strings.Join(cleaned, "\n")
	rawStatements := strings.Split(joined, ";")
	statements := make([]string, 0, len(rawStatements))
	for _, stmt := range rawStatements {
		trimmed := strings.TrimSpace(stmt)
		if trimmed == "" {
			continue
		}
		statements = append(statements, trimmed)
	}
	return statements
}

func managesOwnTransaction(sqlText string) bool {
	statements := parseSQLStatements(sqlText)
	if len(statements) < 2 {
		return false
	}
	firstStmt := strings.ToUpper(strings.Join(strings.Fields(statements[0]), " "))
	lastStmt := strings.ToUpper(strings.Join(strings.Fields(statements[len(statements)-1]), " "))
	isBegin := strings.HasPrefix(firstStmt, "BEGIN") || strings.HasPrefix(firstStmt, "START TRANSACTION")
	isCommit := lastStmt == "COMMIT" || strings.HasPrefix(lastStmt, "COMMIT ")
	return isBegin && isCommit
}

func execMigrationSQL(ctx context.Context, execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, sqlText string) error {
	if strings.TrimSpace(sqlText) == "" {
		return nil
	}
	_, err := execer.ExecContext(ctx, sqlText)
	return err
}

func acquireMigrationsLock(ctx context.Context, conn *sql.Conn) (func(), error) {
	deadline := time.Now().Add(migrationsAdvisoryLockTimeout)
	for {
		var locked bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", migrationsAdvisoryLockKey).Scan(&locked); err != nil {
			return nil, err
		}
		if locked {
			break
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for migration advisory lock after %s", migrationsAdvisoryLockTimeout)
		}
		sleepFor := migrationsAdvisoryLockRetryInterval
		if remaining := time.Until(deadline); remaining < sleepFor {
			sleepFor = remaining
		}
		time.Sleep(sleepFor)
	}
	return func() {
		var unlocked bool
		if err := conn.QueryRowContext(ctx, "SELECT pg_advisory_unlock($1)", migrationsAdvisoryLockKey).Scan(&unlocked); err != nil {
			log.Printf("warning: failed to release migration advisory lock: %v", err)
		} else if !unlocked {
			log.Printf("warning: migration advisory lock was not held by current session")
		}
	}, nil
}

func ApplyMigrations(db *sql.DB, dir string) error {
	ctx := context.Background()
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && isForwardMigrationFile(f.Name()) {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)

	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		return err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	unlock, err := acquireMigrationsLock(ctx, conn)
	if err != nil {
		return err
	}
	defer unlock()

	for _, name := range sqlFiles {
		var exists bool
		err := conn.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)", name).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			log.Printf("skipping already applied migration %s", name)
			continue
		}
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sqlText := string(b)
		if managesOwnTransaction(sqlText) {
			if err := execMigrationSQL(ctx, conn, sqlText); err != nil {
				return err
			}
			if _, err := conn.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
				return err
			}
		} else {
			tx, err := conn.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			if err := execMigrationSQL(ctx, tx, sqlText); err != nil {
				_ = tx.Rollback()
				return err
			}
			if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (name) VALUES ($1)", name); err != nil {
				_ = tx.Rollback()
				return err
			}
			if err := tx.Commit(); err != nil {
				return err
			}
		}
		log.Printf("applied migration %s", name)
	}
	return nil
}

// ApplySeeds applies all .sql files in a seeds directory (lexical order).
// If the directory does not exist, it's a no-op.
func ApplySeeds(db *sql.DB, dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var sqlFiles []string
	for _, f := range files {
		if !f.IsDir() && isForwardMigrationFile(f.Name()) {
			sqlFiles = append(sqlFiles, f.Name())
		}
	}
	sort.Strings(sqlFiles)
	for _, name := range sqlFiles {
		path := filepath.Join(dir, name)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if len(b) > 0 {
			if _, err := db.Exec(string(b)); err != nil {
				return err
			}
		}
		log.Printf("applied seed %s", name)
	}
	return nil
}
