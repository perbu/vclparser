package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/varnish/vclparser/lexer"
	"github.com/varnish/vclparser/parser"
	"github.com/varnish/vclparser/types"
	"github.com/varnish/vclparser/vmod"
)

// TestNamedArgumentMappingRegression tests specific edge cases that previously failed
// due to incorrect parameter mapping in buildCompleteArgumentList
func TestNamedArgumentMappingRegression(t *testing.T) {
	// Create registry with utils VMOD
	registry := vmod.NewRegistry()

	// Create temporary directory for VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	})

	// Create utils.vcc
	utilsVCC := `$Module utils 3 "Utility functions for VCL"
$ABI strict
$Function STRING time_format(STRING format, BOOL local_time = 0, [TIME time])
Format the time according to format.`

	utilsFile := filepath.Join(tmpDir, "utils.vcc")
	err = os.WriteFile(utilsFile, []byte(utilsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write utils.vcc: %v", err)
	}

	// Create std.vcc with real2time function
	stdVCC := `$Module std 3 "Standard library"
$ABI strict
$Function TIME real2time(REAL r, TIME base)`

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(stdVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	// Load VCC files
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC files: %v", err)
	}

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains string
	}{
		{
			name: "time_format with positional format and named time argument",
			vcl: `vcl 4.0;
import utils;
import std;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format("%format", time = std.real2time(-1, now));
}`,
			expectErrors:  false,
			errorContains: "",
		},
		{
			name: "time_format with named format and named time argument",
			vcl: `vcl 4.0;
import utils;
import std;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format(format = "%Y-%m-%d", time = std.real2time(-1, now));
}`,
			expectErrors:  false,
			errorContains: "",
		},
		{
			name: "time_format with positional format and named local_time argument",
			vcl: `vcl 4.0;
import utils;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format("%Y-%m-%d", local_time = true);
}`,
			expectErrors:  false,
			errorContains: "",
		},
		{
			name: "time_format with all named arguments",
			vcl: `vcl 4.0;
import utils;
import std;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format(format = "%Y-%m-%d", local_time = true, time = std.real2time(-1, now));
}`,
			expectErrors:  false,
			errorContains: "",
		},
		{
			name: "time_format with positional format, positional local_time, and named time",
			vcl: `vcl 4.0;
import utils;
import std;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format("%Y-%m-%d", false, time = std.real2time(-1, now));
}`,
			expectErrors:  false,
			errorContains: "",
		},
		{
			name: "time_format with wrong type for named argument",
			vcl: `vcl 4.0;
import utils;

sub vcl_deliver {
    set resp.http.timestamp = utils.time_format("%Y-%m-%d", time = "invalid");
}`,
			expectErrors:  true,
			errorContains: "expected TIME, got STRING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use lexer and parser directly
			l := lexer.New(tt.vcl, "test.vcl")
			p := parser.New(l, tt.vcl, "test.vcl")
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
			}

			symbolTable := types.NewSymbolTable()
			validator := NewVMODValidator(registry, symbolTable)
			errors := validator.Validate(program)

			hasErrors := len(errors) > 0

			if tt.expectErrors != hasErrors {
				if tt.expectErrors {
					t.Errorf("Expected validation errors but got none")
				} else {
					t.Errorf("Expected no validation errors but got: %v", errors)
				}
				return
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' but got: %v", tt.errorContains, errors)
				}
			}
		})
	}
}

// TestOptionalParameterGaps tests edge cases with optional parameters that have gaps
func TestOptionalParameterGaps(t *testing.T) {
	// Create registry with test VMOD
	registry := vmod.NewRegistry()

	// Create temporary directory for VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	})

	// Create test.vcc with complex optional parameter patterns
	testVCC := `$Module test 1 "Test module for optional parameter gaps"
$ABI strict
$Function STRING func_with_gaps(STRING required, [STRING opt1], STRING required2, [BOOL opt2], [INT opt3])
Function with optional parameter gaps.`

	testFile := filepath.Join(tmpDir, "test.vcc")
	err = os.WriteFile(testFile, []byte(testVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write test.vcc: %v", err)
	}

	// Load VCC files
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC files: %v", err)
	}

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains string
	}{
		{
			name: "provide all required args and use named for optional",
			vcl: `vcl 4.0;
import test;

sub vcl_deliver {
    set resp.http.result = test.func_with_gaps("req", "req2", opt2 = true);
}`,
			expectErrors:  true,
			errorContains: "missing required argument 'required2'",
		},
		{
			name: "all named arguments with gaps",
			vcl: `vcl 4.0;
import test;

sub vcl_deliver {
    set resp.http.result = test.func_with_gaps(required = "req", required2 = "req2", opt3 = 42);
}`,
			expectErrors: false,
		},
		{
			name: "mixed positional and named with gaps",
			vcl: `vcl 4.0;
import test;

sub vcl_deliver {
    set resp.http.result = test.func_with_gaps("req", required2 = "req2", opt2 = true);
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use lexer and parser directly
			l := lexer.New(tt.vcl, "test.vcl")
			p := parser.New(l, tt.vcl, "test.vcl")
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
			}

			symbolTable := types.NewSymbolTable()
			validator := NewVMODValidator(registry, symbolTable)
			errors := validator.Validate(program)

			hasErrors := len(errors) > 0

			if tt.expectErrors != hasErrors {
				if tt.expectErrors {
					t.Errorf("Expected validation errors but got none")
				} else {
					t.Errorf("Expected no validation errors but got: %v", errors)
				}
			}

			if tt.expectErrors && tt.errorContains != "" {
				found := false
				for _, err := range errors {
					if contains(err, tt.errorContains) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' but got: %v", tt.errorContains, errors)
				}
			}
		})
	}
}

// contains checks if string s contains substring substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
