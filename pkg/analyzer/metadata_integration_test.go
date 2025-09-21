package analyzer

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

// TestAnalyzerIntegration tests the complete integration through the main Analyzer
func TestAnalyzerIntegration(t *testing.T) {
	tests := []struct {
		name          string
		vclCode       string
		expectErrors  []string
		shouldSucceed bool
	}{
		{
			name: "Complete VCL validation through Analyzer",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    set req.url = "/test";
    return (hash);
}
`,
			shouldSucceed: true,
		},
		{
			name: "VCL with validation errors",
			vclCode: `
vcl 4.0;

sub vcl_recv {
    set beresp.status = 200;       // Variable access error
    return (deliver);              // Return action error
}
`,
			expectErrors: []string{
				"cannot be writed",
				"not allowed",
			},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the VCL code
			program, err := parser.Parse(tt.vclCode, "test.vcl")
			if err != nil && tt.shouldSucceed {
				t.Fatalf("Failed to parse VCL: %v", err)
			}
			if err != nil && !tt.shouldSucceed {
				// Expected parsing to fail
				return
			}

			// Create analyzer
			analyzer := NewAnalyzer(nil)

			// Perform analysis
			errors := analyzer.Analyze(program)

			if tt.shouldSucceed {
				if len(errors) > 0 {
					t.Errorf("Expected no errors but got: %v", errors)
				}
			} else {
				if len(errors) == 0 {
					t.Errorf("Expected errors but got none")
					return
				}

				// Check that expected error messages are present
				for _, expectedError := range tt.expectErrors {
					found := false
					for _, actualError := range errors {
						if strings.Contains(actualError, expectedError) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s' but got: %v", expectedError, errors)
					}
				}
			}
		})
	}
}
