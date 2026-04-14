package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// ===========================
// RemoveLabelFile tests
// ===========================

func TestRemoveLabelFile_EmptyPath(t *testing.T) {
	// Should be a no-op, no panic.
	RemoveLabelFile("")
}

func TestRemoveLabelFile_PathTraversal(t *testing.T) {
	// A path with ".." should be rejected (not remove any file).
	// Create a temp file outside of the label base dir to make sure it survives.
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "should-not-delete.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Try to traverse to the temp file — RemoveLabelFile must refuse.
	RemoveLabelFile("/../../../" + target)

	if _, err := os.Stat(target); os.IsNotExist(err) {
		t.Fatal("RemoveLabelFile deleted a file outside the base directory via path traversal")
	}
}

func TestRemoveLabelFile_NonExistentFile(t *testing.T) {
	// Should not panic when the file doesn't exist.
	RemoveLabelFile("/labels/nonexistent-device-label-file.pdf")
}

func TestRemoveLabelFile_ValidPath(t *testing.T) {
	// Use t.TempDir() as the label base directory to isolate filesystem side effects.
	tmpDir := t.TempDir()
	originalBaseDir := labelBaseDir
	labelBaseDir = tmpDir
	t.Cleanup(func() { labelBaseDir = originalBaseDir })

	labelsDir := filepath.Join(tmpDir, "labels")
	if err := os.MkdirAll(labelsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	labelFile := filepath.Join(labelsDir, "test-device-label.pdf")
	if err := os.WriteFile(labelFile, []byte("label-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	RemoveLabelFile("/labels/test-device-label.pdf")

	if _, err := os.Stat(labelFile); !os.IsNotExist(err) {
		t.Fatalf("expected label file to be removed, but it still exists")
	}
}

// ===========================
// BulkDeleteDevices tests
// ===========================

func newTestService(db *sql.DB) *DeviceAdminService {
	return &DeviceAdminService{db: db}
}

func TestBulkDeleteDevices_AllSucceed(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Use temp dir for label base to isolate filesystem side effects.
	tmpDir := t.TempDir()
	orig := labelBaseDir
	labelBaseDir = tmpDir
	t.Cleanup(func() { labelBaseDir = orig })

	// Create a label file so we can verify cleanup happens.
	labelsDir := filepath.Join(tmpDir, "labels")
	if err := os.MkdirAll(labelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	labelFile := filepath.Join(labelsDir, "dev1.pdf")
	if err := os.WriteFile(labelFile, []byte("label"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	// Device 1: savepoint, delete returning label path, release savepoint
	mock.ExpectExec("SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV001").
		WillReturnRows(sqlmock.NewRows([]string{"label_path"}).AddRow("/labels/dev1.pdf"))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))

	// Device 2: savepoint, delete returning null label path, release savepoint
	mock.ExpectExec("SAVEPOINT device_delete_1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV002").
		WillReturnRows(sqlmock.NewRows([]string{"label_path"}).AddRow(nil))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_1").WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectCommit()

	svc := newTestService(db)
	result, err := svc.BulkDeleteDevices(context.Background(), []string{"DEV001", "DEV002"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 2 {
		t.Errorf("expected 2 deleted, got %d", result.Deleted)
	}
	if result.Failed != 0 {
		t.Errorf("expected 0 failed, got %d", result.Failed)
	}
	if len(result.FailedIDs) != 0 {
		t.Errorf("expected empty FailedIDs, got %v", result.FailedIDs)
	}

	// Verify label file was cleaned up after commit.
	if _, err := os.Stat(labelFile); !os.IsNotExist(err) {
		t.Error("expected label file to be removed after successful commit")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBulkDeleteDevices_PartialFailure_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// Device 1: succeeds
	mock.ExpectExec("SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV001").
		WillReturnRows(sqlmock.NewRows([]string{"label_path"}).AddRow(nil))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))

	// Device 2: not found (no rows)
	mock.ExpectExec("SAVEPOINT device_delete_1").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("NOTFOUND").
		WillReturnRows(sqlmock.NewRows([]string{"label_path"})) // 0 rows = ErrNoRows
	mock.ExpectExec("ROLLBACK TO SAVEPOINT device_delete_1").WillReturnResult(sqlmock.NewResult(0, 0))

	// Device 3: FK violation
	mock.ExpectExec("SAVEPOINT device_delete_2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV_FK").
		WillReturnError(fmt.Errorf("pq: update or delete on table \"devices\" violates foreign key constraint"))
	mock.ExpectExec("ROLLBACK TO SAVEPOINT device_delete_2").WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectCommit()

	svc := newTestService(db)
	result, err := svc.BulkDeleteDevices(context.Background(), []string{"DEV001", "NOTFOUND", "DEV_FK"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", result.Deleted)
	}
	if result.Failed != 2 {
		t.Errorf("expected 2 failed, got %d", result.Failed)
	}
	if len(result.FailedIDs) != 2 {
		t.Errorf("expected 2 FailedIDs, got %v", result.FailedIDs)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBulkDeleteDevices_CommitFailure_NoLabelCleanup(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Use temp dir for label base to verify files are NOT cleaned up on commit failure.
	tmpDir := t.TempDir()
	orig := labelBaseDir
	labelBaseDir = tmpDir
	t.Cleanup(func() { labelBaseDir = orig })

	labelsDir := filepath.Join(tmpDir, "labels")
	if err := os.MkdirAll(labelsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	labelFile := filepath.Join(labelsDir, "dev1.pdf")
	if err := os.WriteFile(labelFile, []byte("label"), 0o644); err != nil {
		t.Fatal(err)
	}

	mock.ExpectBegin()
	mock.ExpectExec("SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV001").
		WillReturnRows(sqlmock.NewRows([]string{"label_path"}).AddRow("/labels/dev1.pdf"))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit().WillReturnError(fmt.Errorf("connection lost"))

	svc := newTestService(db)
	_, err = svc.BulkDeleteDevices(context.Background(), []string{"DEV001"})
	if err == nil {
		t.Fatal("expected commit failure error, got nil")
	}

	// Label file must NOT be cleaned up because the commit failed.
	if _, statErr := os.Stat(labelFile); os.IsNotExist(statErr) {
		t.Error("label file was cleaned up despite commit failure — cleanup must only run after successful commit")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
