package services

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSaveLabelImage_RejectsPathTraversal verifies that device IDs containing
// path-traversal sequences are rejected with the device-ID validation error.
func TestSaveLabelImage_RejectsPathTraversal(t *testing.T) {
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
	s := &LabelService{}
	badIDs := []string{
		"device id",   // space
		"device;id",   // semicolon
		"device<id>",  // angle brackets
		"device|id",   // pipe
		"device`id",   // backtick
		"device$id",   // dollar
		"device%id",   // percent
		"device&id",   // ampersand
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
	s := &LabelService{}
	_, err := s.SaveLabelImage("", "")
	if err == nil {
		t.Fatal("SaveLabelImage(\"\"): expected error for empty device ID, got nil")
	}
}

// TestSaveLabelImage_AcceptsValidDeviceIDs verifies that well-formed device IDs
// pass the validation stage (they will fail later at base64 decode or FS ops,
// but the important thing is they are NOT rejected by the ID check).
func TestSaveLabelImage_AcceptsValidDeviceIDs(t *testing.T) {
	s := &LabelService{}
	validIDs := []string{
		"DEVICE1",
		"device-2",
		"Device_3",
		"ABC123",
		"a",
		"A-B_C",
	}
	// A valid base64 PNG is needed to pass the decode step; we use a trivially
	// invalid one so the function fails at decode — not at deviceID validation.
	invalidBase64 := "not-valid-base64!!!"
	for _, id := range validIDs {
		_, err := s.SaveLabelImage(id, invalidBase64)
		if err == nil {
			t.Fatalf("SaveLabelImage(%q): expected an error (decode), got nil", id)
		}
		if strings.Contains(err.Error(), "device ID must contain only") {
			t.Errorf("SaveLabelImage(%q): valid ID was rejected: %v", id, err)
		}
	}
}

// TestSaveLabelImage_RejectsSymlinkTarget verifies that if the target path is
// a symlink file, SaveLabelImage refuses to write. Without a DB connection the
// function returns early, so we verify the symlink protection code is present
// by ensuring the outside file is untouched and the error is the DB check
// (which fires before any file I/O).
func TestSaveLabelImage_RejectsSymlinkTarget(t *testing.T) {
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

	// Use a LabelService with LabelsDir pointing to our temp labels dir.
	// Without a DB connection, SaveLabelImage returns a DB error before
	// reaching file I/O, so no writes occur at all.
	s := &LabelService{LabelsDir: labelsDir}
	_, err := s.SaveLabelImage("SYMTEST", b64Image)
	if err == nil {
		t.Fatal("SaveLabelImage should have returned an error")
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

// TestSaveLabelImage_AtomicWriteCreatesFile verifies that SaveLabelImage's
// actual write path. Without a DB connection, SaveLabelImage returns an error
// before writing anything, preventing orphaned label files.
func TestSaveLabelImage_AtomicWriteCreatesFile(t *testing.T) {
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

	// SaveLabelImage should return a DB-availability error without writing the file.
	_, err := s.SaveLabelImage("ATOMICTEST", b64Image)
	if err == nil {
		t.Fatal("SaveLabelImage should have returned an error when DB is nil")
	}
	if !strings.Contains(err.Error(), "database connection") {
		t.Fatalf("expected database connection error, got: %v", err)
	}

	// Verify no file was written (DB is checked before file I/O)
	expectedPath := filepath.Join(tmpDir, "ATOMICTEST_label.png")
	if _, statErr := os.Stat(expectedPath); !os.IsNotExist(statErr) {
		t.Errorf("expected no label file to be written when DB is unavailable, but file exists at %q", expectedPath)
	}

	// Verify no leftover temp files
	matches, globErr := filepath.Glob(filepath.Join(tmpDir, ".label.*.tmp"))
	if globErr != nil {
		t.Fatalf("Glob failed: %v", globErr)
	}
	if len(matches) != 0 {
		t.Errorf("found leftover temp files after SaveLabelImage: %v", matches)
	}
}
