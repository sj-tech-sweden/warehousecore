package models

import (
	"encoding/json"
	"testing"
)

func TestJSONStringMarshalNull(t *testing.T) {
	s := JSONString{}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "null" {
		t.Fatalf("expected null, got %s", data)
	}
}

func TestJSONStringMarshalValue(t *testing.T) {
	s := JSONString{}
	s.String = "hello"
	s.Valid = true
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `"hello"` {
		t.Fatalf("expected \"hello\", got %s", data)
	}
}

func TestJSONStringUnmarshalNull(t *testing.T) {
	var s JSONString
	if err := json.Unmarshal([]byte("null"), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Valid {
		t.Fatal("expected Valid=false for null")
	}
	if s.String != "" {
		t.Fatalf("expected empty string, got %q", s.String)
	}
}

func TestJSONStringUnmarshalValue(t *testing.T) {
	var s JSONString
	if err := json.Unmarshal([]byte(`"world"`), &s); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Valid {
		t.Fatal("expected Valid=true")
	}
	if s.String != "world" {
		t.Fatalf("expected \"world\", got %q", s.String)
	}
}

func TestJSONStringUnmarshalInvalid(t *testing.T) {
	var s JSONString
	if err := json.Unmarshal([]byte(`123`), &s); err == nil {
		t.Fatal("expected error for non-string input")
	}
	if s.Valid {
		t.Fatal("expected Valid=false after error")
	}
}

func TestJSONStringPtr(t *testing.T) {
	s := JSONString{}
	if s.Ptr() != nil {
		t.Fatal("expected nil pointer for invalid JSONString")
	}

	s.String = "test"
	s.Valid = true
	p := s.Ptr()
	if p == nil {
		t.Fatal("expected non-nil pointer for valid JSONString")
	}
	if *p != "test" {
		t.Fatalf("expected \"test\", got %q", *p)
	}
}

func TestJSONStringRoundTrip(t *testing.T) {
	type envelope struct {
		Name JSONString `json:"name"`
	}

	original := envelope{}
	original.Name.String = "round-trip"
	original.Name.Valid = true

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !decoded.Name.Valid || decoded.Name.String != "round-trip" {
		t.Fatalf("round-trip failed: got %+v", decoded.Name)
	}
}
