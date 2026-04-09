package services

import (
	"context"
	"testing"
)

// ===========================
// normalizeDeviceIDPrefix tests
// ===========================

func TestNormalizeDeviceIDPrefix_Uppercase(t *testing.T) {
	got := normalizeDeviceIDPrefix("led")
	if got != "LED" {
		t.Fatalf("normalizeDeviceIDPrefix(%q) = %q, want %q", "led", got, "LED")
	}
}

func TestNormalizeDeviceIDPrefix_StripSpecialChars(t *testing.T) {
	got := normalizeDeviceIDPrefix("LED 1!")
	if got != "LED1" {
		t.Fatalf("normalizeDeviceIDPrefix(%q) = %q, want %q", "LED 1!", got, "LED1")
	}
}

func TestNormalizeDeviceIDPrefix_AllStripped(t *testing.T) {
	// A prefix that is all special characters normalizes to empty string.
	got := normalizeDeviceIDPrefix("!@#")
	if got != "" {
		t.Fatalf("normalizeDeviceIDPrefix(%q) = %q, want %q", "!@#", got, "")
	}
}

func TestNormalizeDeviceIDPrefix_AlphaNumeric(t *testing.T) {
	got := normalizeDeviceIDPrefix("ABC123")
	if got != "ABC123" {
		t.Fatalf("normalizeDeviceIDPrefix(%q) = %q, want %q", "ABC123", got, "ABC123")
	}
}

func TestNormalizeDeviceIDPrefix_MixedCase(t *testing.T) {
	got := normalizeDeviceIDPrefix("Le-D_1")
	if got != "LED1" {
		t.Fatalf("normalizeDeviceIDPrefix(%q) = %q, want %q", "Le-D_1", got, "LED1")
	}
}

// ===========================
// deviceIDLikeEscaper tests
// ===========================

func TestDeviceIDLikeEscaper_Percent(t *testing.T) {
	got := deviceIDLikeEscaper.Replace("A%B")
	if got != `A\%B` {
		t.Fatalf("escaping '%%' in prefix: got %q, want %q", got, `A\%B`)
	}
}

func TestDeviceIDLikeEscaper_Underscore(t *testing.T) {
	got := deviceIDLikeEscaper.Replace("A_B")
	if got != `A\_B` {
		t.Fatalf("escaping '_' in prefix: got %q, want %q", got, `A\_B`)
	}
}

func TestDeviceIDLikeEscaper_Backslash(t *testing.T) {
	got := deviceIDLikeEscaper.Replace(`A\B`)
	if got != `A\\B` {
		t.Fatalf(`escaping '\' in prefix: got %q, want %q`, got, `A\\B`)
	}
}

func TestDeviceIDLikeEscaper_NoSpecialChars(t *testing.T) {
	input := "LED1"
	got := deviceIDLikeEscaper.Replace(input)
	if got != input {
		t.Fatalf("escaping plain prefix: got %q, want %q", got, input)
	}
}

func TestDeviceIDLikeEscaper_AllSpecialChars(t *testing.T) {
	got := deviceIDLikeEscaper.Replace(`%_\`)
	want := `\%\_\\`
	if got != want {
		t.Fatalf("escaping all special chars: got %q, want %q", got, want)
	}
}

// ===========================
// DeriveDeviceIDPrefix unit tests (manual-prefix path only; DB path requires
// an actual Postgres connection and is covered by integration tests)
// ===========================

func TestDeriveDeviceIDPrefix_ManualPrefixNormalized(t *testing.T) {
	// When a non-empty manualPrefix is supplied, DeriveDeviceIDPrefix returns
	// early before touching the tx, so it is safe to pass a nil *sql.Tx.
	cases := []struct {
		input string
		want  string
	}{
		{"led", "LED"},
		{"LED 1!", "LED1"},
		{"Abc-123", "ABC123"},
		{"  P1  ", "P1"}, // spaces are stripped; result is P1
	}
	for _, c := range cases {
		got, err := DeriveDeviceIDPrefix(context.Background(), nil, 0, c.input)
		if err != nil {
			t.Errorf("DeriveDeviceIDPrefix(nil, 0, %q) returned unexpected error: %v", c.input, err)
			continue
		}
		if got != c.want {
			t.Errorf("DeriveDeviceIDPrefix(nil, 0, %q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestDeriveDeviceIDPrefix_NilTxWithEmptyPrefix(t *testing.T) {
	// When manualPrefix normalizes to empty (e.g. all special chars) and tx is
	// nil, DeriveDeviceIDPrefix must return an error rather than panicking.
	_, err := DeriveDeviceIDPrefix(context.Background(), nil, 42, "!@#")
	if err == nil {
		t.Fatal("expected error when tx is nil and manualPrefix normalizes to empty, got nil")
	}
}

func TestAllocateDeviceCounter_NilTx(t *testing.T) {
	// AllocateDeviceCounter must return an error when called with a nil tx
	// rather than panicking on the nil pointer dereference.
	_, err := AllocateDeviceCounter(context.Background(), nil, "LED1")
	if err == nil {
		t.Fatal("expected error when tx is nil, got nil")
	}
}

// ===========================
// buildDeviceIDLikePattern tests
// ===========================

func TestBuildDeviceIDLikePattern_NoSpecialChars(t *testing.T) {
	got := buildDeviceIDLikePattern("LED1")
	want := "LED1%"
	if got != want {
		t.Fatalf("buildDeviceIDLikePattern(%q) = %q, want %q", "LED1", got, want)
	}
}

func TestBuildDeviceIDLikePattern_EscapesPercent(t *testing.T) {
	got := buildDeviceIDLikePattern("A%B")
	want := `A\%B%`
	if got != want {
		t.Fatalf("buildDeviceIDLikePattern(%q) = %q, want %q", "A%B", got, want)
	}
}

func TestBuildDeviceIDLikePattern_EscapesUnderscore(t *testing.T) {
	got := buildDeviceIDLikePattern("A_B")
	want := `A\_B%`
	if got != want {
		t.Fatalf("buildDeviceIDLikePattern(%q) = %q, want %q", "A_B", got, want)
	}
}

func TestBuildDeviceIDLikePattern_EscapesBackslash(t *testing.T) {
	got := buildDeviceIDLikePattern(`A\B`)
	want := `A\\B%`
	if got != want {
		t.Fatalf("buildDeviceIDLikePattern(%q) = %q, want %q", `A\B`, got, want)
	}
}

func TestBuildDeviceIDLikePattern_EscapesAll(t *testing.T) {
	got := buildDeviceIDLikePattern(`%_\`)
	want := `\%\_\\%`
	if got != want {
		t.Fatalf("buildDeviceIDLikePattern(%q) = %q, want %q", `%_\`, got, want)
	}
}
