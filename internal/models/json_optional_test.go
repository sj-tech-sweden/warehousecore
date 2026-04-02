package models

import (
	"encoding/json"
	"testing"
)

func TestOptionalAbsent(t *testing.T) {
	type payload struct {
		Field Optional[string] `json:"field"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Field.Set {
		t.Fatal("expected Set=false for absent field")
	}
	if p.Field.Valid {
		t.Fatal("expected Valid=false for absent field")
	}
}

func TestOptionalNull(t *testing.T) {
	type payload struct {
		Field Optional[string] `json:"field"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"field":null}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Field.Set {
		t.Fatal("expected Set=true for explicitly null field")
	}
	if p.Field.Valid {
		t.Fatal("expected Valid=false for null field")
	}
	if p.Field.Ptr() != nil {
		t.Fatal("expected nil pointer for null Optional")
	}
}

func TestOptionalValue(t *testing.T) {
	type payload struct {
		Field Optional[string] `json:"field"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"field":"hello"}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Field.Set {
		t.Fatal("expected Set=true for present field")
	}
	if !p.Field.Valid {
		t.Fatal("expected Valid=true for non-null field")
	}
	if p.Field.Value != "hello" {
		t.Fatalf("expected \"hello\", got %q", p.Field.Value)
	}
}

func TestOptionalPtrReturnsValue(t *testing.T) {
	o := Optional[int]{Value: 42, Set: true, Valid: true}
	p := o.Ptr()
	if p == nil {
		t.Fatal("expected non-nil pointer")
	}
	if *p != 42 {
		t.Fatalf("expected 42, got %d", *p)
	}
}

func TestOptionalPtrNilWhenInvalid(t *testing.T) {
	o := Optional[int]{Value: 0, Set: true, Valid: false}
	if o.Ptr() != nil {
		t.Fatal("expected nil pointer when Valid=false")
	}
}

func TestOptionalIntValue(t *testing.T) {
	type payload struct {
		Count Optional[int] `json:"count"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"count":7}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Count.Valid || p.Count.Value != 7 {
		t.Fatalf("expected count=7, got %+v", p.Count)
	}
}

func TestOptionalBoolValue(t *testing.T) {
	type payload struct {
		Active Optional[bool] `json:"active"`
	}

	var p payload
	if err := json.Unmarshal([]byte(`{"active":true}`), &p); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.Active.Valid || !p.Active.Value {
		t.Fatalf("expected active=true, got %+v", p.Active)
	}
}
