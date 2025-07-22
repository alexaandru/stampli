package main

import (
	"encoding/json"
	"testing"
)

func TestLevelsString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		levels   Levels
		expected string
	}{
		{
			name:     "Empty levels",
			levels:   Levels{},
			expected: "",
		},
		{
			name:     "Single level",
			levels:   Levels{80.0: "#00ff00"},
			expected: "80=#00ff00",
		},
		{
			name:     "Multiple levels sorted",
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "0=#ff0000,70=#ffff00,90=#00ff00",
		},
		{
			name:     "Decimal levels",
			levels:   Levels{85.5: "#00ff00", 72.3: "#ffff00"},
			expected: "72=#ffff00,86=#00ff00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.levels.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestLevelsSet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    Levels
		shouldError bool
	}{
		{
			name:        "Valid single level",
			input:       "80=#00ff00",
			expected:    Levels{80.0: "#00ff00"},
			shouldError: false,
		},
		{
			name:        "Valid multiple levels",
			input:       "90=#00ff00,70=#ffff00,0=#ff0000",
			expected:    Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			shouldError: false,
		},
		{
			name:        "Valid with empty default level",
			input:       "90=#00ff00,=#ff0000",
			expected:    Levels{90.0: "#00ff00", 0.0: "#ff0000"},
			shouldError: false,
		},
		{
			name:        "Valid 3-character hex colors",
			input:       "80=#0f0,60=#ff0",
			expected:    Levels{80.0: "#0f0", 60.0: "#ff0"},
			shouldError: false,
		},
		{
			name:        "Valid with spaces",
			input:       " 80 = #00ff00 , 70 = #ffff00 ",
			expected:    Levels{80.0: "#00ff00", 70.0: "#ffff00"},
			shouldError: false,
		},
		{
			name:        "Empty string",
			input:       "",
			expected:    Levels{},
			shouldError: false,
		},
		{
			name:        "Only commas",
			input:       ",,",
			expected:    Levels{},
			shouldError: false,
		},
		{
			name:        "Invalid format - no equals",
			input:       "80#00ff00",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid format - multiple equals",
			input:       "80=#00ff00=extra",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid level number",
			input:       "abc=#00ff00",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color - no hash",
			input:       "80=00ff00",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color - wrong length",
			input:       "80=#00ff",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color - invalid characters",
			input:       "80=#00gg00",
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Decimal levels",
			input:       "85.5=#00ff00,72.3=#ffff00",
			expected:    Levels{85.5: "#00ff00", 72.3: "#ffff00"},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var levels Levels

			err := levels.Set(tt.input)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Set() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("Set() unexpected error: %v", err)
				return
			}

			if len(levels) != len(tt.expected) {
				t.Errorf("Set() resulted in %d levels, want %d", len(levels), len(tt.expected))
				return
			}

			for level, color := range tt.expected {
				if levels[level] != color {
					t.Errorf("Set() level %f = %q, want %q", level, levels[level], color)
				}
			}
		})
	}
}

func TestLevelsGetColorForCoverage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		coverage float64
		levels   Levels
		expected string
	}{
		{
			name:     "Empty levels - fallback",
			coverage: 75.0,
			levels:   Levels{},
			expected: "#ff0001",
		},
		{
			name:     "Single level - above threshold",
			coverage: 85.0,
			levels:   Levels{80.0: "#00ff00"},
			expected: "#00ff00",
		},
		{
			name:     "Single level - below threshold",
			coverage: 75.0,
			levels:   Levels{80.0: "#00ff00"},
			expected: "#ff0001",
		},
		{
			name:     "Single level - exact match",
			coverage: 80.0,
			levels:   Levels{80.0: "#00ff00"},
			expected: "#00ff00",
		},
		{
			name:     "Multiple levels - high coverage",
			coverage: 95.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#00ff00",
		},
		{
			name:     "Multiple levels - medium coverage",
			coverage: 75.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#ffff00",
		},
		{
			name:     "Multiple levels - low coverage",
			coverage: 25.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#ff0000",
		},
		{
			name:     "Multiple levels - zero coverage",
			coverage: 0.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#ff0000",
		},
		{
			name:     "Multiple levels - exact boundary",
			coverage: 70.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#ffff00",
		},
		{
			name:     "Unordered levels",
			coverage: 75.0,
			levels:   Levels{0.0: "#ff0000", 90.0: "#00ff00", 70.0: "#ffff00"},
			expected: "#ffff00",
		},
		{
			name:     "Decimal levels",
			coverage: 85.5,
			levels:   Levels{85.0: "#00ff00", 70.5: "#ffff00"},
			expected: "#00ff00",
		},
		{
			name:     "Coverage below all levels but has zero level",
			coverage: 5.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#ff0000",
		},
		{
			name:     "Coverage below all levels without zero level",
			coverage: 5.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00"},
			expected: "#ff0001",
		},
		{
			name:     "100% coverage",
			coverage: 100.0,
			levels:   Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			expected: "#00ff00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.levels.GetColorForCoverage(tt.coverage)
			if result != tt.expected {
				t.Errorf("GetColorForCoverage(%f) = %q, want %q",
					tt.coverage, result, tt.expected)
			}
		})
	}
}

func TestLevelsMarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		levels   Levels
		expected string
	}{
		{
			name:     "Empty levels",
			levels:   Levels{},
			expected: `""`,
		},
		{
			name:     "Single level",
			levels:   Levels{80.0: "#00ff00"},
			expected: `"80=#00ff00"`,
		},
		{
			name:     "Zero level",
			levels:   Levels{0.0: "#ff0000"},
			expected: `"0=#ff0000"`,
		},
		{
			name:     "Multiple levels",
			levels:   Levels{90.0: "#00ff00", 0.0: "#ff0000"},
			expected: `"0=#ff0000,90=#00ff00"`,
		},
		{
			name:     "Decimal levels",
			levels:   Levels{85.5: "#00ff00"},
			expected: `"86=#00ff00"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			data, err := json.Marshal(tt.levels)
			if err != nil {
				t.Errorf("MarshalJSON() unexpected error: %v", err)
				return
			}

			// Compare string output directly
			actual := string(data)
			if actual != tt.expected {
				t.Errorf("MarshalJSON() = %q, want %q", actual, tt.expected)
			}
		})
	}
}

func TestLevelsUnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expected    Levels
		shouldError bool
	}{
		{
			name:        "Empty JSON",
			input:       `""`,
			expected:    Levels{},
			shouldError: false,
		},
		{
			name:        "Single level",
			input:       `"80=#00ff00"`,
			expected:    Levels{80.0: "#00ff00"},
			shouldError: false,
		},
		{
			name:        "Zero level",
			input:       `"0=#ff0000"`,
			expected:    Levels{0.0: "#ff0000"},
			shouldError: false,
		},
		{
			name:        "Multiple levels",
			input:       `"90=#00ff00,70=#ffff00,0=#ff0000"`,
			expected:    Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
			shouldError: false,
		},
		{
			name:        "Integer level keys",
			input:       `"80=#00ff00,70=#ffff00"`,
			expected:    Levels{80.0: "#00ff00", 70.0: "#ffff00"},
			shouldError: false,
		},
		{
			name:        "Invalid JSON",
			input:       `"80=#00ff00`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid level key",
			input:       `"abc=#00ff00"`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color",
			input:       `"80=invalid"`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color - no hash",
			input:       `"80=00ff00"`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Invalid hex color - wrong length",
			input:       `"80=#00ff"`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Non-string JSON - number",
			input:       `123`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Non-string JSON - boolean",
			input:       `true`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Non-string JSON - array",
			input:       `["80=#00ff00"]`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Non-string JSON - object",
			input:       `{"80": "#00ff00"}`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Non-string JSON - null",
			input:       `null`,
			expected:    Levels{},
			shouldError: false,
		},
		{
			name:        "Malformed JSON - missing closing quote",
			input:       `"80=#00ff00`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Malformed JSON - extra comma",
			input:       `"80=#00ff00",`,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Empty JSON input",
			input:       ``,
			expected:    nil,
			shouldError: true,
		},
		{
			name:        "Only whitespace JSON",
			input:       `   `,
			expected:    nil,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var levels Levels

			err := json.Unmarshal([]byte(tt.input), &levels)

			if tt.shouldError {
				if err == nil {
					t.Errorf("UnmarshalJSON() expected error but got none")
				}

				return
			}

			if err != nil {
				t.Errorf("UnmarshalJSON() unexpected error: %v", err)
				return
			}

			if len(levels) != len(tt.expected) {
				t.Errorf("UnmarshalJSON() resulted in %d levels, want %d", len(levels), len(tt.expected))
				return
			}

			for level, color := range tt.expected {
				if levels[level] != color {
					t.Errorf("UnmarshalJSON() level %f = %q, want %q", level, levels[level], color)
				}
			}
		})
	}
}

func TestLevelsJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		levels Levels
	}{
		{
			name:   "Empty levels",
			levels: Levels{},
		},
		{
			name:   "Single level",
			levels: Levels{80.0: "#00ff00"},
		},
		{
			name:   "Multiple levels with zero",
			levels: Levels{90.0: "#00ff00", 70.0: "#ffff00", 0.0: "#ff0000"},
		},
		{
			name:   "Decimal levels",
			levels: Levels{85.5: "#00ff00", 72.3: "#ffff00"},
		},
		{
			name:   "3-char hex colors",
			levels: Levels{80.0: "#0f0", 60.0: "#ff0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Marshal to JSON
			data, err := json.Marshal(tt.levels)
			if err != nil {
				t.Errorf("MarshalJSON() unexpected error: %v", err)
				return
			}

			// Unmarshal back to Levels
			var levels Levels

			err = json.Unmarshal(data, &levels)
			if err != nil {
				t.Errorf("UnmarshalJSON() unexpected error: %v", err)
				return
			}

			// Compare
			if len(levels) != len(tt.levels) {
				t.Errorf("Round trip resulted in %d levels, want %d", len(levels), len(tt.levels))
				return
			}

			// For decimal levels test, we need to account for rounding during string conversion
			if tt.name == "Decimal levels" {
				// Check that 85.5 became 86.0 and 72.3 became 72.0
				expectedLevels := Levels{86.0: "#00ff00", 72.0: "#ffff00"}
				for level, color := range expectedLevels {
					if levels[level] != color {
						t.Errorf("Round trip level %f = %q, want %q", level, levels[level], color)
					}
				}
			} else {
				for level, color := range tt.levels {
					if levels[level] != color {
						t.Errorf("Round trip level %f = %q, want %q", level, levels[level], color)
					}
				}
			}
		})
	}
}
