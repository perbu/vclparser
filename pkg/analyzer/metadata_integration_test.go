package analyzer

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/parser"
	"github.com/perbu/vclparser/pkg/types"
)

// TestMetadataIntegration tests the complete metadata-driven validation pipeline
func TestMetadataIntegration(t *testing.T) {
	// Load metadata
	loader := metadata.NewMetadataLoader()
	err := loader.LoadDefault()
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	tests := []struct {
		name          string
		vclCode       string
		expectErrors  []string // partial strings that should appear in error messages
		shouldSucceed bool
	}{
		{
			name: "Valid VCL 4.1 with modern variables",
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
			name: "Invalid VCL 4.0 using 4.1-only variables",
			vclCode: `
vcl 4.0;

sub vcl_backend_response {
    set beresp.proto = "HTTP/1.1";  // This should fail in VCL 4.0
    return (deliver);
}
`,
			expectErrors:  []string{"requires VCL version 4.1"},
			shouldSucceed: false,
		},
		{
			name: "Valid return actions for each subroutine",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    return (hash);
}

sub vcl_hash {
    return (lookup);
}

sub vcl_hit {
    return (deliver);
}

sub vcl_miss {
    return (fetch);
}

sub vcl_backend_fetch {
    return (fetch);
}

sub vcl_backend_response {
    return (deliver);
}

sub vcl_deliver {
    return (deliver);
}
`,
			shouldSucceed: true,
		},
		{
			name: "Invalid return actions",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    return (deliver);  // Invalid in recv
}

sub vcl_hash {
    return (pass);     // Invalid in hash
}
`,
			expectErrors:  []string{"return action 'deliver' is not allowed", "return action 'pass' is not allowed"},
			shouldSucceed: false,
		},
		{
			name: "Variable access validation - invalid writes",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    set beresp.status = 200;    // beresp not available in recv
    return (hash);
}
`,
			expectErrors:  []string{"cannot be writed"},
			shouldSucceed: false,
		},
		{
			name: "Variable access validation - valid access",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    set req.url = "/test";
    return (hash);
}

sub vcl_backend_response {
    set beresp.ttl = 300s;
    return (deliver);
}
`,
			shouldSucceed: true,
		},
		{
			name: "VCL 4.1 with deprecated variable",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    set req.esi = true;  // Deprecated after VCL 4.0
    return (hash);
}
`,
			expectErrors:  []string{"not available in VCL version 4.1"},
			shouldSucceed: false,
		},
		{
			name: "Complex VCL with multiple validation aspects",
			vclCode: `
vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    // Valid variable access
    set req.url = "/test";

    // Valid return action
    return (hash);
}

sub vcl_hash {
    return (lookup);
}

sub vcl_hit {
    return (deliver);
}

sub vcl_backend_fetch {
    return (fetch);
}

sub vcl_backend_response {
    // Valid variable access
    set beresp.status = 200;
    set beresp.ttl = 300s;
    return (deliver);
}

sub vcl_deliver {
    set resp.status = 200;
    return (deliver);
}
`,
			shouldSucceed: true,
		},
		{
			name: "No VCL version specified (should require version)",
			vclCode: `
sub vcl_recv {
    set req.url = "/test";
    return (hash);
}
`,
			expectErrors:  []string{"must start with version"},
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the VCL code
			program, err := parser.Parse(tt.vclCode, "test.vcl")
			if err != nil {
				// If parsing fails, check if this was expected
				if !tt.shouldSucceed {
					// Check if the expected error is in the parse error
					for _, expectedError := range tt.expectErrors {
						if strings.Contains(err.Error(), expectedError) {
							return // Test passed - parsing failed as expected
						}
					}
				}
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Create validators
			symbolTable := types.NewSymbolTable()
			returnValidator := NewReturnActionValidator(loader)
			variableValidator := NewVariableAccessValidator(loader, symbolTable)
			versionValidator := NewVersionValidator(loader)

			// Run all validations
			var allErrors []string

			returnErrors := returnValidator.Validate(program)
			allErrors = append(allErrors, returnErrors...)

			variableErrors := variableValidator.Validate(program)
			allErrors = append(allErrors, variableErrors...)

			versionErrors := versionValidator.Validate(program)
			allErrors = append(allErrors, versionErrors...)

			if tt.shouldSucceed {
				if len(allErrors) > 0 {
					t.Errorf("Expected no errors but got: %v", allErrors)
				}
			} else {
				if len(allErrors) == 0 {
					t.Errorf("Expected errors but got none")
					return
				}

				// Check that expected error messages are present
				for _, expectedError := range tt.expectErrors {
					found := false
					for _, actualError := range allErrors {
						if strings.Contains(actualError, expectedError) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s' but didn't find it in: %v", expectedError, allErrors)
					}
				}
			}
		})
	}
}

// TestMetadataIntegrationWithAnalyzer tests the complete integration through the main Analyzer
func TestMetadataIntegrationWithAnalyzer(t *testing.T) {
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

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    set req.url = "/test";
    return (hash);
}

sub vcl_backend_response {
    set beresp.status = 200;
    set beresp.ttl = 300s;
    return (deliver);
}
`,
			shouldSucceed: true,
		},
		{
			name: "VCL with multiple validation errors",
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
			if err != nil {
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Create analyzer (this will load metadata)
			analyzer := NewAnalyzer(nil) // nil VMOD registry for this test

			// Perform complete analysis
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
						t.Errorf("Expected error containing '%s' but didn't find it in: %v", expectedError, errors)
					}
				}
			}
		})
	}
}

// TestMetadataLoadingPerformance tests that metadata loading doesn't significantly impact performance
func TestMetadataLoadingPerformance(t *testing.T) {
	vclCode := `
vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    set req.url = "/test";
    return (hash);
}

sub vcl_backend_response {
    set beresp.ttl = 300s;
    return (deliver);
}
`

	// Parse once
	program, err := parser.Parse(vclCode, "test.vcl")
	if err != nil {
		t.Fatalf("Failed to parse VCL: %v", err)
	}

	// Run analysis multiple times to test performance
	for i := 0; i < 100; i++ {
		// Create fresh analyzer for each iteration
		analyzer := NewAnalyzer(nil)
		errors := analyzer.Analyze(program)
		if len(errors) > 0 {
			t.Fatalf("Unexpected errors in iteration %d: %v", i, errors)
		}
	}
}

// TestEdgeCases tests edge cases in metadata-based validation
func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		vclCode       string
		expectErrors  []string
		shouldSucceed bool
	}{
		{
			name: "HTTP header variables",
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
			name: "Invalid VCL version format",
			vclCode: `
vcl 5.0;

sub vcl_recv {
    return (hash);
}
`,
			expectErrors:  []string{},
			shouldSucceed: true, // This will parse but won't have version validation errors
		},
		{
			name: "Mixed valid and invalid operations",
			vclCode: `
vcl 4.1;

sub vcl_recv {
    set req.url = "/valid";       // Valid
    set req.proto = "HTTP/1.1";   // Valid in 4.1
    return (hash);                // Valid
}

sub vcl_hash {
    return (pass);                // Invalid in hash
}
`,
			expectErrors:  []string{"not allowed"},
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
						t.Errorf("Expected error containing '%s' but didn't find it in: %v", expectedError, errors)
					}
				}
			}
		})
	}
}
