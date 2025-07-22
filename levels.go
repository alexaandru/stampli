package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Levels represents coverage thresholds and their corresponding colors.
type Levels map[float64]string

var (
	errInvalidLevelNumber = errors.New("invalid level number")
	errInvalidHexColor    = errors.New("invalid hex color")
	errInvalidLevelFormat = errors.New("invalid level format")
)

func (l *Levels) String() string {
	if len(*l) == 0 {
		return ""
	}

	// Sort keys for consistent output.
	keys := make([]float64, 0, len(*l))
	for k := range *l {
		keys = append(keys, k)
	}

	sort.Float64s(keys)

	parts := make([]string, 0, len(keys))

	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%.0f=%s", k, (*l)[k]))
	}

	return strings.Join(parts, ",")
}

// Set implements flag.Value interface.
// Accepts format like "90=#00F,80=#09F,60=#F0F,=#F00" or "0=#F00".
func (l *Levels) Set(value string) error {
	*l = map[float64]string{}

	for part := range strings.SplitSeq(value, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			return fmt.Errorf("%w: %s (expected format: level=color)", errInvalidLevelFormat, part)
		}

		var (
			level float64
			err   error
		)

		levelStr := strings.TrimSpace(kv[0])
		if levelStr == "" {
			level = 0 // Default level.
		} else {
			level, err = strconv.ParseFloat(levelStr, 64)
			if err != nil {
				return fmt.Errorf("%w: %s", errInvalidLevelNumber, levelStr)
			}
		}

		color := strings.TrimSpace(kv[1])
		if !isValidHexColor(color) {
			return fmt.Errorf("%w: %s", errInvalidHexColor, color)
		}

		(*l)[level] = color
	}

	return nil
}

// GetColorForCoverage returns the appropriate color for the given coverage percentage.
func (l *Levels) GetColorForCoverage(coverage float64) string {
	// Sort levels in descending order.
	sortedLevels := make([]float64, 0, len(*l))

	for level := range *l {
		sortedLevels = append(sortedLevels, level)
	}

	sort.Sort(sort.Reverse(sort.Float64Slice(sortedLevels)))

	// Find the first level that coverage meets or exceeds.
	for _, level := range sortedLevels {
		if coverage >= level {
			return (*l)[level]
		}
	}

	// Ultimate fallback.
	return "#ff0001"
}

//nolint:wrapcheck // ok
func (l Levels) MarshalJSON() ([]byte, error) {
	return json.Marshal(l.String())
}

//nolint:wrapcheck // ok
func (l *Levels) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	return l.Set(str)
}
