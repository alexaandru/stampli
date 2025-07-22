package main

import (
	"math"
	"testing"
)

func TestGetOptimalTextColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		hexColor string
		expected string
	}{
		// Light colors (should use black text)
		{
			name:     "White",
			hexColor: "#ffffff",
			expected: "#000000",
		},
		{
			name:     "Light gray",
			hexColor: "#cccccc",
			expected: "#000000",
		},
		{
			name:     "Yellow",
			hexColor: "#ffff00",
			expected: "#000000",
		},
		{
			name:     "Light blue",
			hexColor: "#87ceeb",
			expected: "#000000",
		},
		// Dark colors (should use white text)
		{
			name:     "Black",
			hexColor: "#000000",
			expected: "#ffffff",
		},
		{
			name:     "Dark gray",
			hexColor: "#333333",
			expected: "#ffffff",
		},
		{
			name:     "Dark blue",
			hexColor: "#000080",
			expected: "#ffffff",
		},
		{
			name:     "Red",
			hexColor: "#ff0001",
			expected: "#ffffff",
		},
		// 3-digit hex colors
		{
			name:     "White 3-digit",
			hexColor: "#fff",
			expected: "#000000",
		},
		{
			name:     "Black 3-digit",
			hexColor: "#000",
			expected: "#ffffff",
		},
		{
			name:     "Red 3-digit",
			hexColor: "#f00",
			expected: "#ffffff",
		},
		{
			name:     "Light green 3-digit",
			hexColor: "#0f0",
			expected: "#000000",
		},
		// Without # prefix (edge case handling)
		{
			name:     "White without hash",
			hexColor: "ffffff",
			expected: "#000000",
		},
		{
			name:     "Black without hash",
			hexColor: "000000",
			expected: "#ffffff",
		},
		// Medium colors (boundary testing)
		{
			name:     "Medium gray",
			hexColor: "#808080",
			expected: "#ffffff",
		},
		{
			name:     "Light medium gray",
			hexColor: "#a0a0a0",
			expected: "#ffffff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := getOptimalTextColor(tt.hexColor)
			if result != tt.expected {
				t.Errorf("getOptimalTextColor(%q) = %q, want %q",
					tt.hexColor, result, tt.expected)
			}
		})
	}
}

func TestLuminanceComponent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{
			name:     "Zero",
			input:    0.0,
			expected: 0.0,
		},
		{
			name:     "Small value (linear range)",
			input:    0.02,
			expected: 0.02 / 12.92,
		},
		{
			name:     "Threshold value",
			input:    0.03928,
			expected: 0.03928 / 12.92,
		},
		{
			name:     "Just above threshold",
			input:    0.04,
			expected: math.Pow((0.04+0.055)/1.055, 2.4),
		},
		{
			name:     "Medium value",
			input:    0.5,
			expected: math.Pow((0.5+0.055)/1.055, 2.4),
		},
		{
			name:     "Maximum value",
			input:    1.0,
			expected: math.Pow((1.0+0.055)/1.055, 2.4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := luminanceComponent(tt.input)

			// Use approximate comparison due to floating point precision
			tolerance := 1e-10
			if math.Abs(result-tt.expected) > tolerance {
				t.Errorf("luminanceComponent(%f) = %f, want %f",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidHexColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		color    string
		expected bool
	}{
		// Valid 6-digit hex colors
		{
			name:     "Valid 6-digit lowercase",
			color:    "#ff0000",
			expected: true,
		},
		{
			name:     "Valid 6-digit uppercase",
			color:    "#FF0000",
			expected: true,
		},
		{
			name:     "Valid 6-digit mixed case",
			color:    "#Ff0000",
			expected: true,
		},
		{
			name:     "Valid 6-digit with numbers",
			color:    "#123456",
			expected: true,
		},
		// Valid 3-digit hex colors
		{
			name:     "Valid 3-digit lowercase",
			color:    "#f00",
			expected: true,
		},
		{
			name:     "Valid 3-digit uppercase",
			color:    "#F00",
			expected: true,
		},
		{
			name:     "Valid 3-digit mixed case",
			color:    "#F0a",
			expected: true,
		},
		{
			name:     "Valid 3-digit with numbers",
			color:    "#123",
			expected: true,
		},
		// Invalid - no hash prefix
		{
			name:     "No hash prefix 6-digit",
			color:    "ff0000",
			expected: false,
		},
		{
			name:     "No hash prefix 3-digit",
			color:    "f00",
			expected: false,
		},
		// Invalid - wrong length
		{
			name:     "Too short",
			color:    "#f0",
			expected: false,
		},
		{
			name:     "Too long",
			color:    "#ff00000",
			expected: false,
		},
		{
			name:     "Way too long",
			color:    "#ff0000000",
			expected: false,
		},
		{
			name:     "Length 4",
			color:    "#f000",
			expected: false,
		},
		{
			name:     "Length 5",
			color:    "#f0000",
			expected: false,
		},
		// Invalid - invalid characters
		{
			name:     "Invalid character g",
			color:    "#fg0000",
			expected: false,
		},
		{
			name:     "Invalid character z",
			color:    "#ff000z",
			expected: false,
		},
		{
			name:     "Invalid character space",
			color:    "#ff 000",
			expected: false,
		},
		{
			name:     "Invalid character special",
			color:    "#ff@000",
			expected: false,
		},
		// Edge cases
		{
			name:     "Just hash",
			color:    "#",
			expected: false,
		},
		{
			name:     "Empty string",
			color:    "",
			expected: false,
		},
		{
			name:     "All zeros 6-digit",
			color:    "#000000",
			expected: true,
		},
		{
			name:     "All zeros 3-digit",
			color:    "#000",
			expected: true,
		},
		{
			name:     "All F's 6-digit",
			color:    "#ffffff",
			expected: true,
		},
		{
			name:     "All F's 3-digit",
			color:    "#fff",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := isValidHexColor(tt.color)
			if result != tt.expected {
				t.Errorf("isValidHexColor(%q) = %t, want %t",
					tt.color, result, tt.expected)
			}
		})
	}
}
