package services

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"warehousecore/internal/repository"
)

// ensureNilDB explicitly sets repository.GormDB to nil for the duration of the
// test and restores it afterward. Uses the repository mutex helper so
// concurrent test packages don't race on the global handle.
func ensureNilDB(t *testing.T) {
	t.Helper()
	restore := repository.WithTestGormDB(nil)
	t.Cleanup(restore)
}

// setupMockGormDB creates a go-sqlmock backed *gorm.DB, sets it as
// repository.GormDB, and restores the previous value on cleanup.
// Returns the sqlmock handle so callers can add query expectations.
func setupMockGormDB(t *testing.T) sqlmock.Sqlmock {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		DriverName:           "sqlmock",
		Conn:                 db,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		db.Close()
		t.Fatalf("failed to create gorm DB: %v", err)
	}

	restore := repository.WithTestGormDB(gormDB)
	t.Cleanup(func() {
		restore()
		db.Close()
	})
	return mock
}

// TestSaveLabelImage_RejectsPathTraversal verifies that device IDs containing
// path-traversal sequences are rejected with the device-ID validation error.
func TestSaveLabelImage_RejectsPathTraversal(t *testing.T) {
	ensureNilDB(t)
	s := &LabelService{}
	badIDs := []string{
		"../etc/passwd",
		"../../secret",
		"foo/bar",
		"foo\\bar",
		"..",
		".",
	}
	for _, id := range badIDs {
		_, err := s.SaveLabelImage(id, "")
		if err == nil {
			t.Errorf("SaveLabelImage(%q): expected error for path traversal, got nil", id)
		}
		if err != nil && !strings.Contains(err.Error(), "device ID must contain only") {
			t.Errorf("SaveLabelImage(%q): expected device ID validation error, got: %v", id, err)
		}
	}
}

// TestSaveLabelImage_RejectsDisallowedCharacters verifies that device IDs
// containing characters outside [A-Za-z0-9_-] are rejected.
func TestSaveLabelImage_RejectsDisallowedCharacters(t *testing.T) {
	ensureNilDB(t)
	s := &LabelService{}
	badIDs := []string{
		"device id",    // space
		"device;id",    // semicolon
		"device<id>",   // angle brackets
		"device|id",    // pipe
		"device`id",    // backtick
		"device$id",    // dollar
		"device%id",    // percent
		"device&id",    // ampersand
		"device\x00id", // null byte
	}
	for _, id := range badIDs {
		_, err := s.SaveLabelImage(id, "")
		if err == nil {
			t.Errorf("SaveLabelImage(%q): expected error for disallowed character, got nil", id)
		}
		if err != nil && !strings.Contains(err.Error(), "device ID must contain only") {
			t.Errorf("SaveLabelImage(%q): expected device ID validation error, got %v", id, err)
		}
	}
}

// TestSaveLabelImage_RejectsEmptyDeviceID verifies that an empty device ID is rejected.
func TestSaveLabelImage_RejectsEmptyDeviceID(t *testing.T) {
	ensureNilDB(t)
	s := &LabelService{}
	_, err := s.SaveLabelImage("", "")
	if err == nil {
		t.Fatal("SaveLabelImage(\"\"): expected error for empty device ID, got nil")
	}
}

// TestSaveLabelImage_AcceptsValidDeviceIDs verifies that well-formed device IDs
// pass the validation stage (they will fail later at the DB availability check,
// but the important thing is they are NOT rejected by the ID check).
func TestSaveLabelImage_AcceptsValidDeviceIDs(t *testing.T) {
	ensureNilDB(t)
	s := &LabelService{}
	validIDs := []string{
		"DEVICE1",
		"device-2",
		"Device_3",
		"ABC123",
		"a",
		"A-B_C",
	}
	// Because ensureNilDB(t) forces the DB check to fail first, this test is not
	// asserting the base64 decode path. The image payload is only placeholder
	// input; the key assertion is that valid device IDs are not rejected.
	imageData := "not-valid-base64!!!"
	for _, id := range validIDs {
		_, err := s.SaveLabelImage(id, imageData)
		if err == nil {
			t.Fatalf("SaveLabelImage(%q): expected an error (decode or DB), got nil", id)
		}
		if strings.Contains(err.Error(), "device ID must contain only") {
			t.Errorf("SaveLabelImage(%q): valid ID was rejected: %v", id, err)
		}
	}
}

// TestSaveLabelImage_RejectsSymlinkTarget verifies that if the target path is
// a symlink file, SaveLabelImage refuses to write and does not modify the
// symlink destination. Uses a mock DB so the function reaches the file-I/O
// code path instead of returning early on the nil-DB check.
func TestSaveLabelImage_RejectsSymlinkTarget(t *testing.T) {
	setupMockGormDB(t) // non-nil DB so SaveLabelImage proceeds to file I/O

	// Create a temp labels directory
	tmpDir := t.TempDir()
	labelsDir := filepath.Join(tmpDir, "labels")
	if err := os.MkdirAll(labelsDir, 0755); err != nil {
		t.Fatalf("failed to create test labels dir: %v", err)
	}

	// Create a target file outside the labels directory
	outsideFile := filepath.Join(tmpDir, "outside.txt")
	if err := os.WriteFile(outsideFile, []byte("secret"), 0644); err != nil {
		t.Fatalf("failed to create outside file: %v", err)
	}

	// Create a symlink inside the labels directory pointing outside
	symlinkPath := filepath.Join(labelsDir, "SYMTEST_label.png")
	if err := os.Symlink(outsideFile, symlinkPath); err != nil {
		t.Skipf("cannot create symlinks on this platform: %v", err)
	}

	// Create a minimal valid PNG as base64 (1x1 white pixel PNG)
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	b64Image := base64.StdEncoding.EncodeToString(pngBytes)

	// SaveLabelImage should reject the symlink target
	s := &LabelService{LabelsDir: labelsDir}
	_, err := s.SaveLabelImage("SYMTEST", b64Image)
	if err == nil {
		t.Fatal("SaveLabelImage should have returned an error for symlink target")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected symlink-related error, got: %v", err)
	}

	// Verify the outside file was not modified
	content, err := os.ReadFile(outsideFile)
	if err != nil {
		t.Fatalf("failed to read outside file: %v", err)
	}
	if string(content) != "secret" {
		t.Errorf("outside file was modified: got %q, want %q", string(content), "secret")
	}
}

// TestSaveLabelImage_AtomicWriteCreatesFile verifies SaveLabelImage's actual
// write path by saving a valid PNG and checking the final file exists with the
// complete expected content and no leftover temp files. Uses a mock DB so the
// full code path (write + DB update) is exercised.
func TestSaveLabelImage_AtomicWriteCreatesFile(t *testing.T) {
	mock := setupMockGormDB(t) // non-nil DB so SaveLabelImage proceeds to file I/O + DB update

	tmpDir := t.TempDir()
	s := &LabelService{LabelsDir: tmpDir}

	// 1x1 white pixel PNG
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	b64Image := base64.StdEncoding.EncodeToString(pngBytes)

	// Mock the UPDATE query that SaveLabelImage uses to persist the label path
	mock.ExpectExec(`UPDATE`).
		WillReturnResult(sqlmock.NewResult(0, 1))

	path, err := s.SaveLabelImage("ATOMICTEST", b64Image)
	if err != nil {
		t.Fatalf("SaveLabelImage failed: %v", err)
	}
	if path == "" {
		t.Fatal("SaveLabelImage returned empty path")
	}

	// Verify the label file exists with the correct content
	expectedPath := filepath.Join(tmpDir, "ATOMICTEST_label.png")
	content, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", expectedPath, err)
	}
	if string(content) != string(pngBytes) {
		t.Errorf("saved file content mismatch: got %d bytes, want %d bytes", len(content), len(pngBytes))
	}

	// Verify no leftover temp files
	matches, globErr := filepath.Glob(filepath.Join(tmpDir, ".label.*.tmp"))
	if globErr != nil {
		t.Fatalf("Glob failed: %v", globErr)
	}
	if len(matches) != 0 {
		t.Errorf("found leftover temp files after SaveLabelImage: %v", matches)
	}

	// Verify all sqlmock expectations were met (i.e. the DB UPDATE was executed)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}

// TestSaveLabelImage_RemovesFileWhenDeviceNotFound verifies that when the DB
// UPDATE succeeds but affects 0 rows (device doesn't exist), SaveLabelImage
// returns a "device not found" error and removes the orphaned label file.
func TestSaveLabelImage_RemovesFileWhenDeviceNotFound(t *testing.T) {
	mock := setupMockGormDB(t)

	tmpDir := t.TempDir()
	s := &LabelService{LabelsDir: tmpDir}

	// 1x1 white pixel PNG
	pngBytes := []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
		0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
		0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00,
		0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
	b64Image := base64.StdEncoding.EncodeToString(pngBytes)

	// Mock UPDATE that succeeds but affects 0 rows (device not found)
	mock.ExpectExec(`UPDATE`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	_, err := s.SaveLabelImage("NODEVICE", b64Image)
	if err == nil {
		t.Fatal("SaveLabelImage should have returned an error for non-existent device")
	}
	if !strings.Contains(err.Error(), "device not found") {
		t.Errorf("expected 'device not found' error, got: %v", err)
	}

	// Verify the label file was cleaned up (no orphan on disk)
	labelPath := filepath.Join(tmpDir, "NODEVICE_label.png")
	if _, statErr := os.Stat(labelPath); !os.IsNotExist(statErr) {
		t.Errorf("orphaned label file was not removed: %s", labelPath)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet sqlmock expectations: %v", err)
	}
}
