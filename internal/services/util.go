package services

import "strings"

// nullableString converts a *string to a SQL-compatible value: nil or an empty/
// whitespace-only string becomes SQL NULL; otherwise the trimmed value is returned.
func nullableString(value *string) interface{} {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

// nullableStringPtr is an alias for nullableString kept for backward compatibility
// with call sites that used the name before the helpers were centralized.
func nullableStringPtr(value *string) interface{} {
	return nullableString(value)
}

// nullableText is like nullableString but preserves internal whitespace (only
// the leading/trailing whitespace is used to detect empty values).
func nullableText(value *string) interface{} {
	if value == nil {
		return nil
	}
	if strings.TrimSpace(*value) == "" {
		return nil
	}
	return *value
}

// nullableInt converts a *int to a SQL-compatible value.
func nullableInt(value *int) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

// nullableFloat converts a *float64 to a SQL-compatible value.
func nullableFloat(value *float64) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

// derefFloatOr returns the dereferenced value of v, or def when v is nil.
// Use this when the DB column is NOT NULL (e.g. condition_rating, usage_hours)
// so that a missing optional field is stored as the default rather than NULL.
func derefFloatOr(v *float64, def float64) float64 {
	if v == nil {
		return def
	}
	return *v
}
