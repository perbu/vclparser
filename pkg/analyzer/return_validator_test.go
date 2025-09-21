package analyzer

import (
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/parser"
)

func TestReturnActionValidator_ValidateReturnActions(t *testing.T) {
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
			name: "valid return actions",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					return (hash);
				}
				sub vcl_hash {
					return (lookup);
				}
				sub vcl_deliver {
					return (deliver);
				}
			`,
			expectError: false,
		},
		{
			name: "invalid return action in recv",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					return (lookup); // lookup is only valid in vcl_hash
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid return action in hash",
			vclCode: `vcl 4.1;
				sub vcl_hash {
					return (pass); // pass is not valid in vcl_hash
				}
			`,
			expectError: true,
			errorCount:  1,
		},
		{
			name: "multiple invalid returns",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					return (lookup); // invalid
				}
				sub vcl_hash {
					return (pass); // invalid
				}
			`,
			expectError: true,
			errorCount:  2,
		},
		{
			name: "valid returns with function calls",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					return (synth(404, "Not Found"));
				}
				sub vcl_deliver {
					return (synth(200, "OK"));
				}
			`,
			expectError: false,
		},
		{
			name: "custom subroutine (should be ignored)",
			vclCode: `vcl 4.1;
				sub custom_sub {
					return (lookup); // This should be ignored - not a built-in
				}
				sub vcl_recv {
					return (hash); // This should be validated
				}
			`,
			expectError: false,
		},
		{
			name: "empty returns (should be valid)",
			vclCode: `vcl 4.1;
				sub custom_sub {
					return; // Empty return is always valid
				}
				sub vcl_recv {
					return (hash);
				}
			`,
			expectError: false,
		},
		{
			name: "conditional returns",
			vclCode: `vcl 4.1;
				sub vcl_recv {
					if (req.url ~ "/api/") {
						return (pass);
					} else {
						return (hash);
					}
				}
			`,
			expectError: false,
		},
		{
			name: "invalid conditional return",
			vclCode: `vcl 4.1;
				sub vcl_hash {
					if (req.url ~ "/test") {
						return (pass); // invalid in vcl_hash
					} else {
						return (lookup);
					}
				}
			`,
			expectError: true,
			errorCount:  1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Parse the VCL code
			program, err := parser.Parse(test.vclCode, "test.vcl")
			if err != nil {
				t.Fatalf("Failed to parse VCL: %v", err)
			}

			// Validate return actions
			errors, err := ValidateReturnActions(program, loader)

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

func TestReturnActionValidator_ExtractActionName(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	validator := NewReturnActionValidator(loader)

	tests := []struct {
		name     string
		expr     ast.Expression
		expected string
		hasError bool
	}{
		{
			name:     "simple identifier",
			expr:     &ast.Identifier{Name: "hash"},
			expected: "hash",
			hasError: false,
		},
		{
			name: "function call",
			expr: &ast.CallExpression{
				Function: &ast.Identifier{Name: "synth"},
				Arguments: []ast.Expression{
					&ast.IntegerLiteral{Value: 404},
					&ast.StringLiteral{Value: "Not Found"},
				},
			},
			expected: "synth",
			hasError: false,
		},
		{
			name:     "invalid expression type",
			expr:     &ast.IntegerLiteral{Value: 123},
			expected: "",
			hasError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := validator.extractActionName(test.expr)

			if test.hasError {
				if err == nil {
					t.Error("Expected error, but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if result != test.expected {
					t.Errorf("Expected %s, got %s", test.expected, result)
				}
			}
		})
	}
}

func TestReturnActionValidator_FindReturnStatements(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	validator := NewReturnActionValidator(loader)

	// Create test statements
	returnStmt1 := &ast.ReturnStatement{
		BaseNode: ast.BaseNode{StartPos: lexer.Position{Line: 1}},
		Action:   &ast.Identifier{Name: "hash"},
	}

	returnStmt2 := &ast.ReturnStatement{
		BaseNode: ast.BaseNode{StartPos: lexer.Position{Line: 5}},
		Action:   &ast.Identifier{Name: "pass"},
	}

	ifStmt := &ast.IfStatement{
		Then: &ast.BlockStatement{Statements: []ast.Statement{returnStmt1}},
		Else: &ast.BlockStatement{Statements: []ast.Statement{returnStmt2}},
	}

	statements := []ast.Statement{
		&ast.SetStatement{}, // Non-return statement
		ifStmt,
	}

	returns := validator.findReturnStatements(statements)

	if len(returns) != 2 {
		t.Errorf("Expected 2 return statements, got %d", len(returns))
	}

	// Verify the return statements are the ones we expect
	found1, found2 := false, false
	for _, ret := range returns {
		if ret.StartPos.Line == 1 {
			found1 = true
		}
		if ret.StartPos.Line == 5 {
			found2 = true
		}
	}

	if !found1 || !found2 {
		t.Error("Did not find expected return statements")
	}
}

func TestIsBuiltinSubroutine(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"vcl_recv", true},
		{"vcl_deliver", true},
		{"vcl_backend_fetch", true},
		{"custom_sub", false},
		{"my_subroutine", false},
		{"vcl", false}, // Too short
		{"", false},
	}

	for _, test := range tests {
		result := isBuiltinSubroutine(test.name)
		if result != test.expected {
			t.Errorf("isBuiltinSubroutine(%s) = %v, expected %v", test.name, result, test.expected)
		}
	}
}

func TestExtractMethodName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"vcl_recv", "recv"},
		{"vcl_deliver", "deliver"},
		{"vcl_backend_fetch", "backend_fetch"},
		{"custom_sub", "custom_sub"}, // No vcl_ prefix
		{"vcl", "vcl"},               // Too short
	}

	for _, test := range tests {
		result := extractMethodName(test.input)
		if result != test.expected {
			t.Errorf("extractMethodName(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}
