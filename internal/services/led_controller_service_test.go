package services

import (
	"testing"
)

// ===========================
// normalizeFirmwareType tests
// ===========================

func TestNormalizeFirmwareType_ValidArduino(t *testing.T) {
	ft, ok := normalizeFirmwareType("arduino")
	if !ok || ft != "arduino" {
		t.Fatalf("normalizeFirmwareType(%q) = (%q, %v), want (%q, true)", "arduino", ft, ok, "arduino")
	}
}

func TestNormalizeFirmwareType_ValidEsphome(t *testing.T) {
	ft, ok := normalizeFirmwareType("esphome")
	if !ok || ft != "esphome" {
		t.Fatalf("normalizeFirmwareType(%q) = (%q, %v), want (%q, true)", "esphome", ft, ok, "esphome")
	}
}

func TestNormalizeFirmwareType_CaseInsensitive(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Arduino", "arduino"},
		{"ARDUINO", "arduino"},
		{"ESPHome", "esphome"},
		{"ESPHOME", "esphome"},
		{"EspHome", "esphome"},
	}
	for _, tc := range cases {
		ft, ok := normalizeFirmwareType(tc.input)
		if !ok || ft != tc.want {
			t.Errorf("normalizeFirmwareType(%q) = (%q, %v), want (%q, true)", tc.input, ft, ok, tc.want)
		}
	}
}

func TestNormalizeFirmwareType_TrimWhitespace(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"  arduino  ", "arduino"},
		{"\tesphome\n", "esphome"},
		{" ESPHOME ", "esphome"},
	}
	for _, tc := range cases {
		ft, ok := normalizeFirmwareType(tc.input)
		if !ok || ft != tc.want {
			t.Errorf("normalizeFirmwareType(%q) = (%q, %v), want (%q, true)", tc.input, ft, ok, tc.want)
		}
	}
}

func TestNormalizeFirmwareType_InvalidRejected(t *testing.T) {
	cases := []string{
		"unknown",
		"tasmota",
		"wled",
		"",
		"   ",
		"arduino2",
		"esphome-custom",
	}
	for _, input := range cases {
		ft, ok := normalizeFirmwareType(input)
		if ok || ft != "" {
			t.Errorf("normalizeFirmwareType(%q) = (%q, %v), want (%q, false)", input, ft, ok, "")
		}
	}
}
