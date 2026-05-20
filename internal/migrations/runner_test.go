package migrations

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

func TestManagesOwnTransaction(t *testing.T) {
	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "begin and commit",
			sql:  "BEGIN; SELECT 1; COMMIT;",
			want: true,
		},
		{
			name: "start transaction and commit with comments",
			sql:  "-- header\nSTART TRANSACTION;\nSELECT 1;\n/* note: COMMIT happens below */\nCOMMIT;",
			want: true,
		},
		{
			name: "missing commit",
			sql:  "BEGIN; SELECT 1;",
			want: false,
		},
		{
			name: "commit in comment only",
			sql:  "BEGIN; SELECT 1; -- COMMIT\n",
			want: false,
		},
		{
			name: "empty sql",
			sql:  "  \n\t ",
			want: false,
		},
		{
			name: "non-transactional migration",
			sql:  "ALTER TABLE users ADD COLUMN nickname TEXT;",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := managesOwnTransaction(tt.sql); got != tt.want {
				t.Fatalf("managesOwnTransaction()=%v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplyMigrations_RollsBackFailedMigration(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	dir := t.TempDir()
	name := "001_fail.sql"
	if err := os.WriteFile(filepath.Join(dir, name), []byte("ALTER TABLE test_table ADD COLUMN fail_col INTEGER;\nSELECT bad_syntax"), 0o644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(migrationsAdvisoryLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM schema_migrations WHERE name = \$1\)`).
		WithArgs(name).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectBegin()
	mock.ExpectExec(`ALTER TABLE test_table ADD COLUMN fail_col INTEGER;`).
		WillReturnError(errors.New("boom"))
	mock.ExpectRollback()
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(migrationsAdvisoryLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	err = ApplyMigrations(db, dir)
	if err == nil {
		t.Fatalf("expected migration error, got nil")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}

func TestApplyMigrations_SelfManagedTransactionExecutesWithoutWrapperTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	dir := t.TempDir()
	name := "001_self_tx.sql"
	sqlText := "BEGIN;\nSELECT 1;\nCOMMIT;\n"
	if err := os.WriteFile(filepath.Join(dir, name), []byte(sqlText), 0o644); err != nil {
		t.Fatalf("failed to write migration file: %v", err)
	}

	mock.ExpectExec(`CREATE TABLE IF NOT EXISTS schema_migrations`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(\$1\)`).
		WithArgs(migrationsAdvisoryLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_try_advisory_lock"}).AddRow(true))
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM schema_migrations WHERE name = \$1\)`).
		WithArgs(name).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectExec(`BEGIN;\s*SELECT 1;\s*COMMIT;\s*`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`INSERT INTO schema_migrations \(name\) VALUES \(\$1\)`).
		WithArgs(name).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(\$1\)`).
		WithArgs(migrationsAdvisoryLockKey).
		WillReturnRows(sqlmock.NewRows([]string{"pg_advisory_unlock"}).AddRow(true))

	if err := ApplyMigrations(db, dir); err != nil {
		t.Fatalf("unexpected migration error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sqlmock expectations: %v", err)
	}
}
