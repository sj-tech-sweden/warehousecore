package models

import (
	"testing"
)

func TestIntToNullInt64Nil(t *testing.T) {
	result := IntToNullInt64(nil)
	if result.Valid {
		t.Fatal("expected Valid=false for nil pointer")
	}
}

func TestIntToNullInt64Value(t *testing.T) {
	v := int64(99)
	result := IntToNullInt64(&v)
	if !result.Valid {
		t.Fatal("expected Valid=true for non-nil pointer")
	}
	if result.Int64 != 99 {
		t.Fatalf("expected 99, got %d", result.Int64)
	}
}

func TestIntToNullInt64Zero(t *testing.T) {
	v := int64(0)
	result := IntToNullInt64(&v)
	if !result.Valid {
		t.Fatal("expected Valid=true for zero value pointer")
	}
	if result.Int64 != 0 {
		t.Fatalf("expected 0, got %d", result.Int64)
	}
}

func TestStringToNullStringEmpty(t *testing.T) {
	result := StringToNullString("")
	if result.Valid {
		t.Fatal("expected Valid=false for empty string")
	}
}

func TestStringToNullStringValue(t *testing.T) {
	result := StringToNullString("warehouse")
	if !result.Valid {
		t.Fatal("expected Valid=true for non-empty string")
	}
	if result.String != "warehouse" {
		t.Fatalf("expected \"warehouse\", got %q", result.String)
	}
}

func TestStringToNullStringWhitespace(t *testing.T) {
	result := StringToNullString("  ")
	if !result.Valid {
		t.Fatal("expected Valid=true for whitespace-only string (non-empty)")
	}
}
