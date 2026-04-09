package services

import (
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
	// When a non-empty manualPrefix is supplied, DeriveDeviceIDPrefix should
	// return it normalized (uppercased, [A-Z0-9] only) without touching the DB.
	// We call normalizeDeviceIDPrefix directly because the DB call path is
	// integration-tested; this covers the normalization contract.
	cases := []struct {
		input string
		want  string
	}{
		{"led", "LED"},
		{"LED 1!", "LED1"},
		{"Abc-123", "ABC123"},
		{"  P1  ", "P1"}, // trimmed then normalized
	}
	for _, c := range cases {
		got := normalizeDeviceIDPrefix(c.input)
		if got != c.want {
			t.Errorf("normalizeDeviceIDPrefix(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}
