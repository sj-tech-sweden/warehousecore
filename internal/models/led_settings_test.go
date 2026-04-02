package models

import (
	"testing"
)

func TestDefaultLEDJobHighlightSettings(t *testing.T) {
	d := DefaultLEDJobHighlightSettings()
	if d == nil {
		t.Fatal("expected non-nil defaults")
	}
	if d.Mode != "all_bins" {
		t.Fatalf("expected mode=all_bins, got %q", d.Mode)
	}
	if d.Required.Color == "" {
		t.Fatal("expected non-empty required color")
	}
	if d.NonRequired.Color == "" {
		t.Fatal("expected non-empty non-required color")
	}
}

func TestNormalizeModeValid(t *testing.T) {
	s := &LEDJobHighlightSettings{
		Mode:        "required_only",
		Required:    LEDAppearance{Color: "#FF0000", Pattern: "solid", Intensity: 100, Speed: 500},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "blink", Intensity: 50, Speed: 300},
	}
	s.Normalize(nil)
	if s.Mode != "required_only" {
		t.Fatalf("expected mode=required_only, got %q", s.Mode)
	}
}

func TestNormalizeModeInvalidFallsBackToDefault(t *testing.T) {
	s := &LEDJobHighlightSettings{
		Mode:        "invalid_mode",
		Required:    LEDAppearance{Color: "#FF0000", Pattern: "solid", Intensity: 100, Speed: 500},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "blink", Intensity: 50, Speed: 300},
	}
	s.Normalize(nil)
	defaults := DefaultLEDJobHighlightSettings()
	if s.Mode != defaults.Mode {
		t.Fatalf("expected mode=%q, got %q", defaults.Mode, s.Mode)
	}
}

func TestNormalizeEmptyColorFallsBackToDefault(t *testing.T) {
	defaults := DefaultLEDJobHighlightSettings()
	s := &LEDJobHighlightSettings{
		Mode: "all_bins",
		Required: LEDAppearance{
			Color:     "",
			Pattern:   "solid",
			Intensity: 100,
			Speed:     500,
		},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "blink", Intensity: 50, Speed: 300},
	}
	s.Normalize(defaults)
	if s.Required.Color != defaults.Required.Color {
		t.Fatalf("expected required color=%q, got %q", defaults.Required.Color, s.Required.Color)
	}
}

func TestNormalizeEmptyPatternFallsBackToDefault(t *testing.T) {
	defaults := DefaultLEDJobHighlightSettings()
	s := &LEDJobHighlightSettings{
		Mode: "all_bins",
		Required: LEDAppearance{
			Color:     "#FF0000",
			Pattern:   "",
			Intensity: 100,
			Speed:     500,
		},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "solid", Intensity: 50, Speed: 300},
	}
	s.Normalize(defaults)
	if s.Required.Pattern != defaults.Required.Pattern {
		t.Fatalf("expected required pattern=%q, got %q", defaults.Required.Pattern, s.Required.Pattern)
	}
}

func TestNormalizeZeroIntensityFallsBackToDefault(t *testing.T) {
	defaults := DefaultLEDJobHighlightSettings()
	s := &LEDJobHighlightSettings{
		Mode: "all_bins",
		Required: LEDAppearance{
			Color:     "#FF0000",
			Pattern:   "solid",
			Intensity: 0,
			Speed:     500,
		},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "blink", Intensity: 50, Speed: 300},
	}
	s.Normalize(defaults)
	if s.Required.Intensity != defaults.Required.Intensity {
		t.Fatalf("expected required intensity=%d, got %d", defaults.Required.Intensity, s.Required.Intensity)
	}
}

func TestNormalizeZeroSpeedFallsBackToDefault(t *testing.T) {
	defaults := DefaultLEDJobHighlightSettings()
	s := &LEDJobHighlightSettings{
		Mode: "all_bins",
		Required: LEDAppearance{
			Color:     "#FF0000",
			Pattern:   "solid",
			Intensity: 100,
			Speed:     0,
		},
		NonRequired: LEDAppearance{Color: "#00FF00", Pattern: "blink", Intensity: 50, Speed: 300},
	}
	s.Normalize(defaults)
	if s.Required.Speed != defaults.Required.Speed {
		t.Fatalf("expected required speed=%d, got %d", defaults.Required.Speed, s.Required.Speed)
	}
}

func TestNormalizeFullyPopulatedSettingsUnchanged(t *testing.T) {
	s := &LEDJobHighlightSettings{
		Mode: "required_only",
		Required: LEDAppearance{
			Color:     "#AABBCC",
			Pattern:   "breathe",
			Intensity: 200,
			Speed:     800,
		},
		NonRequired: LEDAppearance{
			Color:     "#112233",
			Pattern:   "blink",
			Intensity: 100,
			Speed:     400,
		},
	}
	s.Normalize(nil)
	if s.Mode != "required_only" {
		t.Fatalf("mode changed unexpectedly: %q", s.Mode)
	}
	if s.Required.Color != "#AABBCC" {
		t.Fatalf("required color changed: %q", s.Required.Color)
	}
	if s.Required.Pattern != "breathe" {
		t.Fatalf("required pattern changed: %q", s.Required.Pattern)
	}
	if s.Required.Intensity != 200 {
		t.Fatalf("required intensity changed: %d", s.Required.Intensity)
	}
	if s.Required.Speed != 800 {
		t.Fatalf("required speed changed: %d", s.Required.Speed)
	}
}

func TestNormalizeWithCustomDefaults(t *testing.T) {
	customDefaults := &LEDJobHighlightSettings{
		Mode: "required_only",
		Required: LEDAppearance{
			Color:     "#FFFFFF",
			Pattern:   "blink",
			Intensity: 128,
			Speed:     600,
		},
		NonRequired: LEDAppearance{
			Color:     "#000000",
			Pattern:   "solid",
			Intensity: 64,
			Speed:     200,
		},
	}

	s := &LEDJobHighlightSettings{
		Mode: "invalid",
		Required: LEDAppearance{
			Color:     "",
			Pattern:   "",
			Intensity: 0,
			Speed:     0,
		},
		NonRequired: LEDAppearance{},
	}
	s.Normalize(customDefaults)

	if s.Mode != "required_only" {
		t.Fatalf("expected mode=required_only from custom defaults, got %q", s.Mode)
	}
	if s.Required.Color != "#FFFFFF" {
		t.Fatalf("expected required color=#FFFFFF from custom defaults, got %q", s.Required.Color)
	}
	if s.Required.Intensity != 128 {
		t.Fatalf("expected required intensity=128 from custom defaults, got %d", s.Required.Intensity)
	}
}
