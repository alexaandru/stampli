package main

import (
	"math"
	"strconv"
	"strings"
)

// getOptimalTextColor determines whether to use white or
// black text based on background color.
//
//nolint:errcheck,mnd // hex colors are validated when set
func getOptimalTextColor(hexColor string) string {
	hexColor = strings.TrimPrefix(hexColor, "#")

	// Expand 3-digit hex to 6-digit.
	if len(hexColor) == 3 {
		hexColor = string([]byte{hexColor[0], hexColor[0], hexColor[1], hexColor[1], hexColor[2], hexColor[2]})
	}

	// Parse RGB components.
	r, _ := strconv.ParseInt(hexColor[0:2], 16, 0)
	g, _ := strconv.ParseInt(hexColor[2:4], 16, 0)
	b, _ := strconv.ParseInt(hexColor[4:6], 16, 0)

	// Calculate relative luminance using the formula from WCAG
	// https://www.w3.org/WAI/GL/wiki/Relative_luminance
	rLum := luminanceComponent(float64(r) / 255.0)
	gLum := luminanceComponent(float64(g) / 255.0)
	bLum := luminanceComponent(float64(b) / 255.0)

	// Use white text on dark backgrounds, black text on light backgrounds.
	luminance := 0.2126*rLum + 0.7152*gLum + 0.0722*bLum
	if luminance > 0.5 {
		return "#000000"
	}

	return "#ffffff"
}

// luminanceComponent calculates the luminance component for a single RGB channel.
//
//nolint:mnd // ok
func luminanceComponent(c float64) float64 {
	if c <= 0.03928 {
		return c / 12.92
	}

	return math.Pow((c+0.055)/1.055, 2.4)
}

func isValidHexColor(color string) bool {
	if !strings.HasPrefix(color, "#") {
		return false
	}

	color = color[1:]
	if len(color) != 3 && len(color) != 6 {
		return false
	}

	for _, c := range color {
		//nolint:staticcheck // ok
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}
