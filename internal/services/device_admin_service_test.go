package services

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

// ===========================
// RemoveLabelFile tests
// ===========================

func TestRemoveLabelFile_EmptyPath(t *testing.T) {
	// Should be a no-op, no panic.
	RemoveLabelFile("")
}

func TestRemoveLabelFile_PathTraversal(t *testing.T) {
	// Use a temp label base dir and create a target file outside it to verify traversal is blocked.
	baseDir := t.TempDir()
	originalBaseDir := labelBaseDir
	labelBaseDir = baseDir
	t.Cleanup(func() { labelBaseDir = originalBaseDir })

	outsideDir := t.TempDir()
	target := filepath.Join(outsideDir, "should-not-delete.txt")
	if err := os.WriteFile(target, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}

	relTarget, err := filepath.Rel(baseDir, target)
	if err != nil {
		t.Fatal(err)
	}
	relTarget = filepath.ToSlash(relTarget)
	if !strings.HasPrefix(relTarget, "..") {
		t.Fatalf("expected traversal path from %q to %q, got %q", baseDir, target, relTarget)
	}

	// Try to traverse from labelBaseDir to the outside file — RemoveLabelFile must refuse.
	RemoveLabelFile("/" + relTarget)

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
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_1").WillReturnResult(sqlmock.NewResult(0, 0))

	// Device 3: FK violation
	mock.ExpectExec("SAVEPOINT device_delete_2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV_FK").
		WillReturnError(fmt.Errorf("pq: update or delete on table \"devices\" violates foreign key constraint"))
	mock.ExpectExec("ROLLBACK TO SAVEPOINT device_delete_2").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_2").WillReturnResult(sqlmock.NewResult(0, 0))

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

	// Verify error messages are sanitized (no raw DB/driver details)
	if reason, ok := result.FailedErrors["NOTFOUND"]; !ok {
		t.Error("expected FailedErrors to contain NOTFOUND")
	} else if reason != "device not found" {
		t.Errorf("expected 'device not found' for NOTFOUND, got: %s", reason)
	}
	if reason, ok := result.FailedErrors["DEV_FK"]; !ok {
		t.Error("expected FailedErrors to contain DEV_FK")
	} else if reason != "internal error deleting device" {
		t.Errorf("expected 'internal error deleting device' for DEV_FK (generic error), got: %s", reason)
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

func TestBulkDeleteDevices_FKViolation_UserFriendlyMessage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// Device with FK violation: savepoint, delete returns pq error 23503, rollback + release
	mock.ExpectExec("SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("DELETE FROM devices WHERE deviceID = \\$1 RETURNING label_path").
		WithArgs("DEV_LINKED").
		WillReturnError(&pq.Error{Code: "23503", Message: "update or delete on table \"devices\" violates foreign key constraint"})
	mock.ExpectExec("ROLLBACK TO SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec("RELEASE SAVEPOINT device_delete_0").WillReturnResult(sqlmock.NewResult(0, 0))

	// No commit expected — deleted == 0, so tx is rolled back by defer
	mock.ExpectRollback()

	svc := newTestService(db)
	result, err := svc.BulkDeleteDevices(context.Background(), []string{"DEV_LINKED"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Deleted != 0 {
		t.Errorf("expected 0 deleted, got %d", result.Deleted)
	}
	if result.Failed != 1 {
		t.Errorf("expected 1 failed, got %d", result.Failed)
	}
	if len(result.FailedIDs) != 1 || result.FailedIDs[0] != "DEV_LINKED" {
		t.Errorf("expected FailedIDs=[DEV_LINKED], got %v", result.FailedIDs)
	}

	// Verify the error message is user-friendly (from the pq 23503 branch)
	reason, ok := result.FailedErrors["DEV_LINKED"]
	if !ok {
		t.Fatal("expected FailedErrors to contain DEV_LINKED")
	}
	if !strings.Contains(reason, "still linked to cases, jobs, or history entries") {
		t.Errorf("expected user-friendly FK violation message, got: %s", reason)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ===========================
// BulkUpdateDevices tests
// ===========================

func TestBulkUpdateDevices_StatusNormalization_FreeToInStorage(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// "free" should be normalized to "in_storage"
	mock.ExpectExec("UPDATE devices SET status = \\$1 WHERE deviceID = \\$2").
		WithArgs("in_storage", "DEV001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	svc := newTestService(db)
	status := "free"
	result, err := svc.BulkUpdateDevices(context.Background(), []string{"DEV001"}, &BulkUpdateDeviceInput{
		Status: &status,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Updated != 1 {
		t.Errorf("expected 1 updated, got %d", result.Updated)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBulkUpdateDevices_NoFieldsToUpdate(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newTestService(db)
	_, err = svc.BulkUpdateDevices(context.Background(), []string{"DEV001"}, &BulkUpdateDeviceInput{})
	if err == nil {
		t.Fatal("expected error for no fields to update, got nil")
	}
	if !strings.Contains(err.Error(), "no fields to update") {
		t.Errorf("expected 'no fields to update' error, got: %v", err)
	}
}

func TestBulkUpdateDevices_NilInput(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newTestService(db)
	_, err = svc.BulkUpdateDevices(context.Background(), []string{"DEV001"}, nil)
	if err == nil {
		t.Fatal("expected error for nil input, got nil")
	}
	if !strings.Contains(err.Error(), "input cannot be nil") {
		t.Errorf("expected 'input cannot be nil' error, got: %v", err)
	}
}

func TestBulkUpdateDevices_PerDeviceUpdateFailure_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectBegin()
	// First device succeeds
	mock.ExpectExec("UPDATE devices SET status = \\$1 WHERE deviceID = \\$2").
		WithArgs("retired", "DEV001").
		WillReturnResult(sqlmock.NewResult(0, 1))
	// Second device fails — should trigger rollback
	mock.ExpectExec("UPDATE devices SET status = \\$1 WHERE deviceID = \\$2").
		WithArgs("retired", "DEV002").
		WillReturnError(fmt.Errorf("connection lost"))
	mock.ExpectRollback()

	svc := newTestService(db)
	status := "retired"
	_, err = svc.BulkUpdateDevices(context.Background(), []string{"DEV001", "DEV002"}, &BulkUpdateDeviceInput{
		Status: &status,
	})
	if err == nil {
		t.Fatal("expected error on device update failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to update device DEV002") {
		t.Errorf("expected device-specific error, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestBulkUpdateDevices_EmptyStatusRejected(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	svc := newTestService(db)
	emptyStatus := "   "
	_, err = svc.BulkUpdateDevices(context.Background(), []string{"DEV001"}, &BulkUpdateDeviceInput{
		Status: &emptyStatus,
	})
	if err == nil {
		t.Fatal("expected error for empty status, got nil")
	}
	if !strings.Contains(err.Error(), "status cannot be empty") {
		t.Errorf("expected 'status cannot be empty' error, got: %v", err)
	}
}
