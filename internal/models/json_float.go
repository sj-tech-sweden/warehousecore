package models

import (
	"bytes"
	"database/sql"
	"encoding/json"
)

// JSONFloat64 wraps sql.NullFloat64 but marshals to `null` or a primitive number for JSON clients.
type JSONFloat64 struct {
	sql.NullFloat64
}

// MarshalJSON converts the value to either a float64 literal or null.
func (f JSONFloat64) MarshalJSON() ([]byte, error) {
	if !f.Valid {
		return []byte("null"), nil
	}
	return json.Marshal(f.Float64)
}

// UnmarshalJSON populates the float from a JSON number or null.
func (f *JSONFloat64) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) || len(trimmed) == 0 {
		f.Float64 = 0
		f.Valid = false
		return nil
	}

	if err := json.Unmarshal(trimmed, &f.Float64); err != nil {
		f.Float64 = 0
		f.Valid = false
		return err
	}

	f.Valid = true
	return nil
}

// Ptr returns a pointer to the float64 when valid, otherwise nil.
func (f JSONFloat64) Ptr() *float64 {
	if !f.Valid {
		return nil
	}
	val := f.Float64
	return &val
}
