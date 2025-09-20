package parser

import (
	"strconv"
	"strings"
)

// Duration units mapping to seconds (similar to Varnish's VNUM_duration_unit)
var durationUnits = map[string]float64{
	"ms": 0.001,    // milliseconds
	"s":  1,        // seconds
	"m":  60,       // minutes
	"h":  3600,     // hours
	"d":  86400,    // days
	"w":  604800,   // weeks (7 * 24 * 3600)
	"y":  31536000, // years (365 * 24 * 3600)
}

// IsDurationUnit checks if the given string is a valid duration unit
// This validates units supported by Varnish: ms, s, m, h, d, w, y
func IsDurationUnit(unit string) bool {
	_, exists := durationUnits[unit]
	return exists
}

// ParseDuration parses a duration string (e.g., "30s", "1.5h") and returns the value in seconds
// This mimics Varnish's duration parsing behavior
func ParseDuration(durationStr string) (float64, error) {
	// Check for units in order of length (longest first) to avoid "ms" being matched as "m"
	units := []string{"ms", "s", "m", "h", "d", "w", "y"}

	var numPart, unitPart string
	for _, unit := range units {
		if strings.HasSuffix(durationStr, unit) {
			numPart = strings.TrimSuffix(durationStr, unit)
			unitPart = unit
			break
		}
	}

	if unitPart == "" {
		// No valid unit found, return 0 without error (matches old behavior)
		return 0, nil
	}

	// Parse the numeric part
	value, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0, err
	}

	// Convert to seconds
	multiplier := durationUnits[unitPart]
	return value * multiplier, nil
}

// GetSupportedDurationUnits returns a slice of all supported duration units
func GetSupportedDurationUnits() []string {
	units := make([]string, 0, len(durationUnits))
	for unit := range durationUnits {
		units = append(units, unit)
	}
	return units
}

// ValidateDurationString checks if a complete duration string is valid
// Returns true for strings like "30s", "1.5h", "0ms", etc.
func ValidateDurationString(durationStr string) bool {
	if durationStr == "" {
		return false
	}

	// Check for units in order of length (longest first) to avoid "ms" being matched as "m"
	units := []string{"ms", "s", "m", "h", "d", "w", "y"}

	var numPart, unitPart string
	for _, unit := range units {
		if strings.HasSuffix(durationStr, unit) {
			numPart = strings.TrimSuffix(durationStr, unit)
			unitPart = unit
			break
		}
	}

	// Must have a valid unit
	if unitPart == "" {
		return false
	}

	// Must have a valid number part
	if numPart == "" {
		return false
	}

	// Parse the numeric part to validate it
	_, err := strconv.ParseFloat(numPart, 64)
	return err == nil
}
