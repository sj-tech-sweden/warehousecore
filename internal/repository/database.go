// Package repository provides PostgreSQL database connection for WarehouseCore
package repository

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"warehousecore/config"
	"warehousecore/internal/migrations"

	_ "github.com/lib/pq" // PostgreSQL driver
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Common errors
var (
	ErrNotFound = errors.New("not found")
)

func closeSQLDBWithLog(sqlDB *sql.DB, reason string) {
	if sqlDB == nil {
		return
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("failed to close database after %s: %v", reason, err)
	}
}

// DB holds the database connection pool (sql.DB)
var DB *sql.DB

// GormDB holds the GORM database connection for auth and models
var GormDB *gorm.DB

// InitDatabase initializes the PostgreSQL database connection
func InitDatabase(cfg *config.Config) error {
	// Build PostgreSQL DSN
	dsn := buildPostgresDSN(cfg)

	// Open sql.DB for direct SQL queries
	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// PostgreSQL connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(30 * time.Minute)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		closeSQLDBWithLog(sqlDB, "failed ping")
		return fmt.Errorf("failed to ping database: %w", err)
	}

	DB = sqlDB
	log.Printf("PostgreSQL database connection established: %s@%s:%s/%s",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)

	// Optionally run SQL migrations and seeds on startup. Controlled by
	// env var MIGRATE_ON_STARTUP (unset/empty defaults to disabled). The migrations directory
	// can be overridden with MIGRATIONS_DIR (default: "migrations").
	migrateOnStartup := strings.TrimSpace(os.Getenv("MIGRATE_ON_STARTUP"))
	if migrateOnStartup == "" {
		log.Println("MIGRATE_ON_STARTUP is not set; startup migrations are disabled by default. Set MIGRATE_ON_STARTUP=true to enable.")
	}
	if strings.EqualFold(migrateOnStartup, "true") {
		// Determine migrations directory (allow override via MIGRATIONS_DIR env)
		migrationsDir := os.Getenv("MIGRATIONS_DIR")
		if migrationsDir == "" {
			migrationsDir = "migrations"
		}
		// Use the repo-relative path; if running from project root this is fine.
		absDir, _ := filepath.Abs(migrationsDir)
		log.Printf("Running SQL migrations from %s", absDir)
		if err := migrations.ApplyMigrations(sqlDB, migrationsDir); err != nil {
			closeSQLDBWithLog(sqlDB, "failed migrations")
			DB = nil
			return fmt.Errorf("apply migrations: %w", err)
		}
		// Apply all seeds in migrations/seeds. Seed files are expected to be
		// idempotent so they are safe to execute on populated databases.
		seedsDir := filepath.Join(migrationsDir, "seeds")
		if err := migrations.ApplySeeds(sqlDB, seedsDir); err != nil {
			closeSQLDBWithLog(sqlDB, "failed seeds")
			DB = nil
			return fmt.Errorf("apply seeds: %w", err)
		}
		log.Println("Migrations and startup seeds applied")
	}

	// Initialize GORM with PostgreSQL driver
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: false,
		PrepareStmt:            true,
		CreateBatchSize:        100,
		Logger:                 logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		closeSQLDBWithLog(sqlDB, "failed gorm initialization")
		DB = nil
		return fmt.Errorf("failed to initialize GORM: %w", err)
	}

	GormDB = gormDB
	log.Println("GORM PostgreSQL connection established successfully")

	return nil
}

// buildPostgresDSN creates the PostgreSQL connection string
func buildPostgresDSN(cfg *config.Config) string {
	host := cfg.Database.Host
	if host == "" {
		host = "localhost"
	}

	port := cfg.Database.Port
	if port == "" {
		port = "5432"
	}

	dbName := cfg.Database.Name
	if dbName == "" {
		dbName = "warehousecore"
	}

	user := cfg.Database.User
	if user == "" {
		user = "warehousecore"
	}

	password := cfg.Database.Password

	sslMode := cfg.Database.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslMode,
	)
}

// CloseDatabase closes the database connection properly
func CloseDatabase() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

// GetDB returns the GORM database connection
func GetDB() *gorm.DB {
	return GormDB
}

// GetSQLDB returns the raw SQL database connection
func GetSQLDB() *sql.DB {
	return DB
}

// apiKeyPepper is an application-level secret used to HMAC API key hashes.
// Set via API_KEY_PEPPER env var; defaults to a built-in value so the app
// works out of the box, but operators SHOULD set their own pepper.
var apiKeyPepper = func() string {
	if v := os.Getenv("API_KEY_PEPPER"); v != "" {
		return v
	}
	log.Println("WARNING: API_KEY_PEPPER is not set – using default pepper. Set API_KEY_PEPPER env var for production use.")
	return "warehousecore-default-api-key-pepper"
}()

// HashAPIKey creates a keyed HMAC-SHA256 hex digest of an API key.
// The pepper prevents rainbow-table attacks even if the database is leaked.
func HashAPIKey(key string) string {
	mac := hmac.New(sha256.New, []byte(apiKeyPepper))
	mac.Write([]byte(key))
	return hex.EncodeToString(mac.Sum(nil))
}
