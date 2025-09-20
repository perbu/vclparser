package vcc

import (
	"strings"
	"testing"
)

func TestParseBracketParameters(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectError  bool
		expectedOpt  bool
		expectedName string
		expectedType VCCType
		expectedDef  string
	}{
		{
			name:         "simple optional parameter",
			input:        "[REAL br_q = 1]",
			expectError:  false,
			expectedOpt:  true,
			expectedName: "br_q",
			expectedType: TypeReal,
			expectedDef:  "1",
		},
		{
			name:         "optional boolean parameter",
			input:        "[BOOL large_window = 0]",
			expectError:  false,
			expectedOpt:  true,
			expectedName: "large_window",
			expectedType: TypeBool,
			expectedDef:  "0",
		},
		{
			name:         "optional parameter without name",
			input:        "[INT quality = 6]",
			expectError:  false,
			expectedOpt:  true,
			expectedName: "quality",
			expectedType: TypeInt,
			expectedDef:  "6",
		},
		{
			name:         "optional ENUM parameter",
			input:        "[ENUM {GENERIC, UTF8, FONT} mode = GENERIC]",
			expectError:  false,
			expectedOpt:  true,
			expectedName: "mode",
			expectedType: TypeEnum,
			expectedDef:  "GENERIC",
		},
		{
			name:         "required parameter (no brackets)",
			input:        "STRING s",
			expectError:  false,
			expectedOpt:  false,
			expectedName: "s",
			expectedType: TypeString,
			expectedDef:  "",
		},
		{
			name:         "named parameter without brackets",
			input:        "BYTES buf_size = 32768",
			expectError:  false,
			expectedOpt:  true,
			expectedName: "buf_size",
			expectedType: TypeBytes,
			expectedDef:  "32768",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := &Parser{}
			param, err := parser.parseParameter(test.input)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if param.Optional != test.expectedOpt {
				t.Errorf("Expected Optional=%v, got %v", test.expectedOpt, param.Optional)
			}

			if param.Name != test.expectedName {
				t.Errorf("Expected Name=%s, got %s", test.expectedName, param.Name)
			}

			if param.Type != test.expectedType {
				t.Errorf("Expected Type=%v, got %v", test.expectedType, param.Type)
			}

			if param.DefaultValue != test.expectedDef {
				t.Errorf("Expected DefaultValue=%s, got %s", test.expectedDef, param.DefaultValue)
			}
		})
	}
}

func TestSplitParametersWithBrackets(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:  "mixed parameters",
			input: "ENUM {BR, GZIP, BOTH, NONE} encoding, [REAL br_q = 1], [REAL gzip_q = 1]",
			expected: []string{
				"ENUM {BR, GZIP, BOTH, NONE} encoding",
				" [REAL br_q = 1]",
				" [REAL gzip_q = 1]",
			},
		},
		{
			name:  "complex brotli function",
			input: "ENUM {BR, GZIP, BOTH, NONE} encoding, [REAL br_q = 1], [REAL gzip_q = 1], BYTES buf_size = 32768, ENUM {GENERIC, UTF8, FONT} mode = GENERIC",
			expected: []string{
				"ENUM {BR, GZIP, BOTH, NONE} encoding",
				" [REAL br_q = 1]",
				" [REAL gzip_q = 1]",
				" BYTES buf_size = 32768",
				" ENUM {GENERIC, UTF8, FONT} mode = GENERIC",
			},
		},
		{
			name:  "nested brackets and braces",
			input: "[ENUM {A, B} type = A], [ENUM {X, Y} other = X]",
			expected: []string{
				"[ENUM {A, B} type = A]",
				" [ENUM {X, Y} other = X]",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parser := &Parser{}
			result := parser.splitParameters(test.input)

			if len(result) != len(test.expected) {
				t.Errorf("Expected %d parts, got %d: %v", len(test.expected), len(result), result)
				return
			}

			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("Part %d: expected %q, got %q", i, expected, result[i])
				}
			}
		})
	}
}

// TestParseParameterTokensWithBrackets is skipped for now due to the need for a mock lexer
// This functionality will be tested through the integration test below

func TestRealWorldBrotliFunction(t *testing.T) {
	// Test parsing a simple VCC function signature with brackets
	vccContent := `$Module brotli 3 "Brotli"
$Function VOID compress([INT quality = 6])`

	parser := NewParser(strings.NewReader(vccContent))

	module, err := parser.Parse()
	if err != nil {
		t.Logf("Parse errors: %v", err)

		// Let's try without brackets to see if basic parsing works
		vccContent2 := `$Module brotli 3 "Brotli"
$Function VOID compress(INT quality)`
		parser2 := NewParser(strings.NewReader(vccContent2))
		module2, err2 := parser2.Parse()
		if err2 != nil {
			t.Fatalf("Even basic parsing fails: %v", err2)
		}
		if len(module2.Functions) == 1 {
			t.Logf("Basic parsing works, issue is with bracket syntax")
		}

		t.Fatalf("Failed to parse VCC content: %v", err)
	}

	if len(module.Functions) != 1 {
		t.Fatalf("Expected 1 function, got %d", len(module.Functions))
	}

	function := module.Functions[0]
	if function.Name != "compress" {
		t.Errorf("Expected function name 'compress', got %s", function.Name)
	}

	if len(function.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(function.Parameters))
	}

	param := function.Parameters[0]
	if !param.Optional {
		t.Errorf("Parameter should be optional")
	}
	if param.DefaultValue != "6" {
		t.Errorf("Expected default value '6', got '%s'", param.DefaultValue)
	}
	if param.Name != "quality" {
		t.Errorf("Expected parameter name 'quality', got '%s'", param.Name)
	}
}
