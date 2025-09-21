package analyzer

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/parser"
	"github.com/perbu/vclparser/pkg/types"
)

func TestVariableAccessValidator_ValidateVariableAccesses(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	tests := []struct {
		name        string
		vclCode     string
		expectError bool
		errorCount  int
	}{
		{
			name: "valid variable reads",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					if (req.method == "GET") {
						set req.url = "/test";
					}
				}
			`,
			expectError: false,
		},
		{
			name: "invalid variable write in wrong context",
			vclCode: `vcl 4.1;
				sub vcl_backend_fetch {
					set req.url = "/test"; // req.url not writable in backend context
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid variable read in wrong context",
			vclCode: `vcl 4.1;
				sub vcl_deliver {
					if (bereq.method == "GET") { // bereq not readable in deliver
						return (deliver);
					}
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "valid header access",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					set req.http.useragent = "test";
				}
			`,
			expectError: false,
		},
		{
			name: "invalid header write in wrong context",
			vclCode: `vcl 4.1;
				sub vcl_backend_fetch {
					set req.http.host = "example.com"; // req.http not writable in backend context
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "valid unset operation",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					unset req.http.useragent;
				}
			`,
			expectError: false,
		},
		{
			name: "invalid unset in wrong context",
			vclCode: `vcl 4.1;
				sub vcl_backend_fetch {
					unset req.http.cookie; // req.http not unsetable in backend context
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "valid backend variable access",
			vclCode: `vcl 4.1;
				sub vcl_backend_fetch {
					set bereq.method = "GET";
					if (bereq.url ~ "/api/") {
						return (fetch);
					}
				}
			`,
			expectError: false,
		},
		{
			name: "invalid backend variable in client context",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					if (bereq.method == "POST") { // bereq not readable in recv
						return (pass);
					}
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "multiple variable access errors",
			vclCode: `vcl 4.1;
				sub vcl_backend_fetch {
					set req.url = "/test";        // invalid write - req.url not writable in backend
					if (resp.status == 200) {    // invalid read - resp not readable in backend
						return (fetch);
					}
				}
			`,
			expectError: true,
			errorCount:  2,
		},
		{
			name: "custom subroutine (should be ignored)",
			vclCode: `vcl 4.1;
				sub custom_sub {
					set req.url = "/test"; // Should be ignored - not built-in
				}
				sub vcl_recv {
					set req.url = "/valid"; // Should be validated and pass
				}
			`,
			expectError: false,
		},
		{
			name: "complex member access",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					if (client.ip ~ "192.168.1.0/24") {
						return (pass);
					}
				}
			`,
			expectError: false,
		},
		{
			name: "assignment expression",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					set req.url = req.url + "?test=1"; // Simple assignment with binary expression
				}
			`,
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the VCL code
			program, err := parser.Parse(test.vclCode, "test.vcl")
			if err != nil {
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Validate variable accesses
			errors, err := ValidateVariableAccesses(program, loader)

			if test.expectError {
				if err == nil {
					t.Errorf("Expected validation errors, but got none")
				}
				if test.errorCount > 0 && len(errors) != test.errorCount {
					t.Errorf("Expected %d errors, got %d: %v", test.errorCount, len(errors), errors)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation errors, got: %v", err)
				}
			}
		})
	}
}

func TestVariableAccessValidator_ExtractVariableName(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	symbolTable := types.NewSymbolTable()
	validator := NewVariableAccessValidator(loader, symbolTable)

	tests := []struct {
		name     string
		vclExpr  string
		expected string
	}{
		{
			name:     "simple identifier",
			vclExpr:  "req",
			expected: "req",
		},
		{
			name:     "member expression",
			vclExpr:  "req.url",
			expected: "req.url",
		},
		{
			name:     "nested member expression",
			vclExpr:  "req.http.host",
			expected: "req.http.host",
		},
		{
			name:     "complex header access",
			vclExpr:  "req.http.useragent",
			expected: "req.http.useragent",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse a simple expression to get the AST
			vclCode := fmt.Sprintf(`vcl 4.1; sub vcl_recv { if (%s) { return (hash); } }`, test.vclExpr)
			program, err := parser.Parse(vclCode, "test.vcl")
			if err != nil {
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Extract the expression from the if statement
			subDecl := program.Declarations[0].(*ast.SubDecl)
			ifStmt := subDecl.Body.Statements[0].(*ast.IfStatement)

			result := validator.extractVariableName(ifStmt.Condition)
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
}

func TestVariableAccessValidator_ExtractMemberVariableName(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	symbolTable := types.NewSymbolTable()
	validator := NewVariableAccessValidator(loader, symbolTable)

	tests := []struct {
		name     string
		vclExpr  string
		expected string
	}{
		{
			name:     "req.url",
			vclExpr:  "req.url",
			expected: "req.url",
		},
		{
			name:     "req.http.host",
			vclExpr:  "req.http.host",
			expected: "req.http.host",
		},
		{
			name:     "beresp.http.content_length",
			vclExpr:  "beresp.http.content_length",
			expected: "beresp.http.content_length",
		},
		{
			name:     "client.ip",
			vclExpr:  "client.ip",
			expected: "client.ip",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse a simple expression to get the AST
			vclCode := fmt.Sprintf(`vcl 4.1; sub vcl_recv { if (%s) { return (hash); } }`, test.vclExpr)
			program, err := parser.Parse(vclCode, "test.vcl")
			if err != nil {
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Extract the member expression from the if statement
			subDecl := program.Declarations[0].(*ast.SubDecl)
			ifStmt := subDecl.Body.Statements[0].(*ast.IfStatement)

			if memberExpr, ok := ifStmt.Condition.(*ast.MemberExpression); ok {
				result := validator.extractMemberVariableName(memberExpr)
				if result != test.expected {
					t.Errorf("Expected %s, got %s", test.expected, result)
				}
			} else {
				t.Errorf("Expected MemberExpression, got %T", ifStmt.Condition)
			}
		})
	}
}
