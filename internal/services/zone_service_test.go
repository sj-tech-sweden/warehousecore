package services

import (
	"testing"
)

func TestGeneratePrefixShort(t *testing.T) {
	// Names with 3 or fewer letters return the letters directly (uppercased)
	cases := []struct {
		input    string
		expected string
	}{
		{"ABC", "ABC"},
		{"ab", "AB"},
		{"x", "X"},
	}
	for _, c := range cases {
		got := generatePrefix(c.input)
		if got != c.expected {
			t.Fatalf("generatePrefix(%q) = %q, want %q", c.input, got, c.expected)
		}
	}
}

func TestGeneratePrefixConsonants(t *testing.T) {
	// "Weidelbach" → consonants W, D, L, B, C, H → first 3 = "WDL"
	got := generatePrefix("Weidelbach")
	if got != "WDL" {
		t.Fatalf("generatePrefix(\"Weidelbach\") = %q, want \"WDL\"", got)
	}
}

func TestGeneratePrefixStripsSpecialChars(t *testing.T) {
	// "A-B C" → letters only: "ABC" → len=3, return as-is
	got := generatePrefix("A-B C")
	if got != "ABC" {
		t.Fatalf("generatePrefix(\"A-B C\") = %q, want \"ABC\"", got)
	}
}

func TestGeneratePrefixFewConsonantsFallbackToFirst3(t *testing.T) {
	// "aeiou" → consonants = none → fallback to first 3 letters: "AEI"
	got := generatePrefix("aeiou")
	if got != "AEI" {
		t.Fatalf("generatePrefix(\"aeiou\") = %q, want \"AEI\"", got)
	}
}

func TestGeneratePrefixNumbers(t *testing.T) {
	// "123abc" → strip non-alpha → "ABC" → 3 letters, return as-is
	got := generatePrefix("123abc")
	if got != "ABC" {
		t.Fatalf("generatePrefix(\"123abc\") = %q, want \"ABC\"", got)
	}
}

func TestGetTypePrefixKnown(t *testing.T) {
	cases := []struct {
		zoneType string
		expected string
	}{
		{"warehouse", "LGR"},
		{"rack", "RG"},
		{"gitterbox", "GB"},
		{"shelf", "F"},
	}
	for _, c := range cases {
		got := getTypePrefix(c.zoneType)
		if got != c.expected {
			t.Fatalf("getTypePrefix(%q) = %q, want %q", c.zoneType, got, c.expected)
		}
	}
}

func TestGetTypePrefixUnknown(t *testing.T) {
	got := getTypePrefix("unknown_type")
	if got != "OT" {
		t.Fatalf("getTypePrefix(\"unknown_type\") = %q, want \"OT\"", got)
	}
}

func TestGetTypePrefixEmpty(t *testing.T) {
	got := getTypePrefix("")
	if got != "OT" {
		t.Fatalf("getTypePrefix(\"\") = %q, want \"OT\"", got)
	}
}
