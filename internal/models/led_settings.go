package models

// LEDAppearance defines color and animation for LED commands
type LEDAppearance struct {
	Color     string `json:"color"`
	Pattern   string `json:"pattern"`
	Intensity uint8  `json:"intensity"`
	Speed     int    `json:"speed"` // milliseconds per cycle for animated patterns
}

// LEDJobHighlightSettings controls how bins are highlighted for job packing
type LEDJobHighlightSettings struct {
	Mode        string        `json:"mode"` // all_bins or required_only
	Required    LEDAppearance `json:"required"`
	NonRequired LEDAppearance `json:"non_required"`
}

// DefaultLEDJobHighlightSettings provides safe defaults when nothing is configured
func DefaultLEDJobHighlightSettings() *LEDJobHighlightSettings {
	return &LEDJobHighlightSettings{
		Mode: "all_bins",
		Required: LEDAppearance{
			Color:     "#00FF00",
			Pattern:   "solid",
			Intensity: 220,
			Speed:     1200,
		},
		NonRequired: LEDAppearance{
			Color:     "#FF0000",
			Pattern:   "solid",
			Intensity: 160,
			Speed:     1200,
		},
	}
}

// Normalize ensures the settings have sensible defaults and valid values.
func (s *LEDJobHighlightSettings) Normalize(defaults *LEDJobHighlightSettings) {
	if defaults == nil {
		defaults = DefaultLEDJobHighlightSettings()
	}

	if s.Mode != "all_bins" && s.Mode != "required_only" {
		s.Mode = defaults.Mode
	}

	s.Required = s.normalizeAppearance(s.Required, defaults.Required)
	s.NonRequired = s.normalizeAppearance(s.NonRequired, defaults.NonRequired)
}

func (s *LEDJobHighlightSettings) normalizeAppearance(app LEDAppearance, fallback LEDAppearance) LEDAppearance {
	if app.Color == "" {
		app.Color = fallback.Color
	}
	if app.Pattern == "" {
		app.Pattern = fallback.Pattern
	}
	if app.Intensity == 0 {
		app.Intensity = fallback.Intensity
	}
	// Ensure speed is non-negative. When zero, animations can use fallback speed.
	if app.Speed <= 0 {
		app.Speed = fallback.Speed
	}
	return app
}
