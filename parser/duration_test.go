package parser

import (
	"testing"
)

func TestIsDurationUnit(t *testing.T) {
	tests := []struct {
		unit     string
		expected bool
	}{
		// Valid units
		{"s", true},
		{"m", true},
		{"h", true},
		{"d", true},
		{"w", true},
		{"ms", true},
		{"y", true},

		// Invalid units
		{"ns", false}, // ns not supported (unlike old code)
		{"us", false}, // us not supported (unlike old code)
		{"sec", false},
		{"min", false},
		{"hour", false},
		{"day", false},
		{"week", false},
		{"year", false},
		{"", false},
		{"x", false},
		{"ss", false},
	}

	for _, test := range tests {
		result := IsDurationUnit(test.unit)
		if result != test.expected {
			t.Errorf("IsDurationUnit(%q) = %v, expected %v", test.unit, result, test.expected)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		hasError bool
	}{
		// Basic integer durations
		{"30s", 30, false},
		{"5m", 300, false},      // 5 * 60 = 300
		{"2h", 7200, false},     // 2 * 3600 = 7200
		{"1d", 86400, false},    // 1 * 86400 = 86400
		{"1w", 604800, false},   // 1 * 604800 = 604800
		{"1y", 31536000, false}, // 1 * 31536000 = 31536000

		// Milliseconds
		{"500ms", 0.5, false},
		{"1000ms", 1, false},

		// Float durations
		{"1.5s", 1.5, false},
		{"0.5h", 1800, false}, // 0.5 * 3600 = 1800
		{"2.5m", 150, false},  // 2.5 * 60 = 150

		// Edge cases
		{"0s", 0, false},
		{"0.0s", 0, false},

		// Invalid inputs
		{"", 0, false},       // No unit, returns 0 without error
		{"10", 0, false},     // No unit, returns 0 without error
		{"s", 0, true},       // No number
		{"10x", 0, false},    // Invalid unit, returns 0 without error
		{"abc", 0, false},    // Invalid string without unit, returns 0 without error
		{"10.5.5s", 0, true}, // Invalid number format
	}

	for _, test := range tests {
		result, err := ParseDuration(test.input)

		if test.hasError {
			if err == nil {
				t.Errorf("ParseDuration(%q) expected error but got none", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("ParseDuration(%q) = %v, expected %v", test.input, result, test.expected)
			}
		}
	}
}

func TestValidateDurationString(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Valid duration strings
		{"30s", true},
		{"1.5h", true},
		{"500ms", true},
		{"0s", true},
		{"10m", true},
		{"7d", true},
		{"2w", true},
		{"1y", true},

		// Invalid duration strings
		{"", false},
		{"10", false},
		{"s", false},
		{"10x", false},
		{"abc", false},
		{"10.5.5s", false},
		{"-5s", true}, // Negative durations are valid numbers technically
	}

	for _, test := range tests {
		result := ValidateDurationString(test.input)
		if result != test.expected {
			t.Errorf("ValidateDurationString(%q) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestGetSupportedDurationUnits(t *testing.T) {
	units := GetSupportedDurationUnits()

	// Check that we have the expected number of units
	expectedCount := 7 // ms, s, m, h, d, w, y
	if len(units) != expectedCount {
		t.Errorf("GetSupportedDurationUnits() returned %d units, expected %d", len(units), expectedCount)
	}

	// Check that all expected units are present
	expectedUnits := []string{"ms", "s", "m", "h", "d", "w", "y"}
	for _, expected := range expectedUnits {
		found := false
		for _, unit := range units {
			if unit == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetSupportedDurationUnits() missing expected unit: %s", expected)
		}
	}
}
