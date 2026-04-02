package models

import (
	"encoding/json"
	"testing"
)

func TestJSONFloat64MarshalNull(t *testing.T) {
	f := JSONFloat64{}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Fatalf("expected null, got %s", data)
	}
}

func TestJSONFloat64MarshalValue(t *testing.T) {
	f := JSONFloat64{}
	f.Float64 = 3.14
	f.Valid = true
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "3.14" {
		t.Fatalf("expected 3.14, got %s", data)
	}
}

func TestJSONFloat64UnmarshalNull(t *testing.T) {
	var f JSONFloat64
	if err := json.Unmarshal([]byte("null"), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f.Valid {
		t.Fatal("expected Valid=false for null")
	}
	if f.Float64 != 0 {
		t.Fatalf("expected 0, got %v", f.Float64)
	}
}

func TestJSONFloat64UnmarshalValue(t *testing.T) {
	var f JSONFloat64
	if err := json.Unmarshal([]byte("2.718"), &f); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !f.Valid {
		t.Fatal("expected Valid=true")
	}
	if f.Float64 != 2.718 {
		t.Fatalf("expected 2.718, got %v", f.Float64)
	}
}

func TestJSONFloat64UnmarshalInvalid(t *testing.T) {
	var f JSONFloat64
	if err := json.Unmarshal([]byte(`"notanumber"`), &f); err == nil {
		t.Fatal("expected error for non-number input")
	}
	if f.Valid {
		t.Fatal("expected Valid=false after error")
	}
}

func TestJSONFloat64Ptr(t *testing.T) {
	f := JSONFloat64{}
	if f.Ptr() != nil {
		t.Fatal("expected nil pointer for invalid JSONFloat64")
	}

	f.Float64 = 42.0
	f.Valid = true
	p := f.Ptr()
	if p == nil {
		t.Fatal("expected non-nil pointer for valid JSONFloat64")
	}
	if *p != 42.0 {
		t.Fatalf("expected 42.0, got %v", *p)
	}
}

func TestJSONFloat64RoundTrip(t *testing.T) {
	type envelope struct {
		Value JSONFloat64 `json:"value"`
	}

	original := envelope{}
	original.Value.Float64 = 1.5
	original.Value.Valid = true

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.Value.Valid || decoded.Value.Float64 != 1.5 {
		t.Fatalf("round-trip failed: got %+v", decoded.Value)
	}
}
