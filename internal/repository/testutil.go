package repository

import (
	"database/sql"
	"sync"

	"gorm.io/gorm"
)

// testDBMu serializes cross-package mutations of the global database handles
// (DB and GormDB) during test execution. Packages run in parallel by default
// under `go test ./...`; acquiring this mutex before swapping a global handle
// prevents data races without requiring -count=1 or -parallel=1 flags.
var testDBMu sync.Mutex

// WithTestSQLDB replaces the global SQL DB handle with db for the duration of
// a test. It acquires testDBMu and returns a cleanup func that restores the
// original handle and releases the mutex. Register it with t.Cleanup.
func WithTestSQLDB(db *sql.DB) func() {
	testDBMu.Lock()
	orig := DB
	DB = db
	return func() {
		DB = orig
		testDBMu.Unlock()
	}
}

// WithTestGormDB replaces the global GORM DB handle with db for the duration
// of a test. It acquires testDBMu and returns a cleanup func that restores the
// original handle and releases the mutex. Register it with t.Cleanup.
func WithTestGormDB(db *gorm.DB) func() {
	testDBMu.Lock()
	orig := GormDB
	GormDB = db
	return func() {
		GormDB = orig
		testDBMu.Unlock()
	}
}

// WithTestDatabases atomically replaces both global DB handles. Use when a
// test needs to nil-out (or replace) both sql.DB and gorm.DB together.
// Register the returned cleanup func with t.Cleanup.
func WithTestDatabases(sqlDB *sql.DB, gormDB *gorm.DB) func() {
	testDBMu.Lock()
	origSQL := DB
	origGorm := GormDB
	DB = sqlDB
	GormDB = gormDB
	return func() {
		DB = origSQL
		GormDB = origGorm
		testDBMu.Unlock()
	}
}
