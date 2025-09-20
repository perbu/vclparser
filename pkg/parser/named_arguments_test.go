package parser

import (
	"testing"

	ast2 "github.com/varnish/vclparser/pkg/ast"
	"github.com/varnish/vclparser/pkg/lexer"
)

func TestNamedArgumentParsing(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		description string
		checkArgs   func(t *testing.T, callExpr *ast2.CallExpression)
	}{
		{
			name: "Pure positional arguments",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(NAME, ";", ":");
}`,
			wantErr:     false,
			description: "All arguments are positional - baseline test",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 3 {
					t.Errorf("Expected 3 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 0 {
					t.Errorf("Expected 0 named arguments, got %d", len(callExpr.NamedArguments))
				}
			},
		},
		{
			name: "Pure named arguments",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(type = NAME, separator = ";", name_case = LOWER);
}`,
			wantErr:     false,
			description: "All arguments are named",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 0 {
					t.Errorf("Expected 0 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 3 {
					t.Errorf("Expected 3 named arguments, got %d", len(callExpr.NamedArguments))
				}

				expectedNames := []string{"type", "separator", "name_case"}
				for _, name := range expectedNames {
					if _, exists := callExpr.NamedArguments[name]; !exists {
						t.Errorf("Expected named argument '%s' not found", name)
					}
				}
			},
		},
		{
			name: "Mixed positional and named arguments",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(NAME, ";", name_case = LOWER);
}`,
			wantErr:     false,
			description: "First two arguments positional, third named",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 2 {
					t.Errorf("Expected 2 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 1 {
					t.Errorf("Expected 1 named argument, got %d", len(callExpr.NamedArguments))
				}
				if _, exists := callExpr.NamedArguments["name_case"]; !exists {
					t.Errorf("Expected named argument 'name_case' not found")
				}
			},
		},
		{
			name: "Named argument with function call value",
			input: `vcl 4.0;
sub test {
	xbody.regsub("pattern", "replacement", max = std.integer(bereq.http.max, 0));
}`,
			wantErr:     false,
			description: "Named argument value is a function call",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 2 {
					t.Errorf("Expected 2 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 1 {
					t.Errorf("Expected 1 named argument, got %d", len(callExpr.NamedArguments))
				}

				maxArg, exists := callExpr.NamedArguments["max"]
				if !exists {
					t.Errorf("Expected named argument 'max' not found")
					return
				}

				// Check that the named argument value is a function call
				if _, ok := maxArg.(*ast2.CallExpression); !ok {
					t.Errorf("Expected named argument 'max' to be a CallExpression, got %T", maxArg)
				}
			},
		},
		{
			name: "Single named argument",
			input: `vcl 4.0;
sub test {
	utils.time_format(format = "%Y-%m-%d");
}`,
			wantErr:     false,
			description: "Single named argument only",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 0 {
					t.Errorf("Expected 0 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 1 {
					t.Errorf("Expected 1 named argument, got %d", len(callExpr.NamedArguments))
				}
				if _, exists := callExpr.NamedArguments["format"]; !exists {
					t.Errorf("Expected named argument 'format' not found")
				}
			},
		},
		{
			name: "Complex named arguments with different types",
			input: `vcl 4.0;
sub test {
	s3.verify(access_key_id = "KEY", secret_key = "SECRET", debug = true);
}`,
			wantErr:     false,
			description: "Named arguments with string and boolean values",
			checkArgs: func(t *testing.T, callExpr *ast2.CallExpression) {
				if len(callExpr.Arguments) != 0 {
					t.Errorf("Expected 0 positional arguments, got %d", len(callExpr.Arguments))
				}
				if len(callExpr.NamedArguments) != 3 {
					t.Errorf("Expected 3 named arguments, got %d", len(callExpr.NamedArguments))
				}

				expectedNames := []string{"access_key_id", "secret_key", "debug"}
				for _, name := range expectedNames {
					if _, exists := callExpr.NamedArguments[name]; !exists {
						t.Errorf("Expected named argument '%s' not found", name)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			hasErrors := len(p.Errors()) > 0

			if hasErrors != tt.wantErr {
				t.Errorf("Test case: %s", tt.description)
				if tt.wantErr {
					t.Errorf("Expected parse error but got none")
				} else {
					t.Errorf("Unexpected parse errors:")
					for _, err := range p.Errors() {
						t.Errorf("  - %s", err)
					}
				}
				return
			}

			// If parsing succeeded and we have a check function, validate the AST
			if !hasErrors && !tt.wantErr && tt.checkArgs != nil && program != nil {
				// Find the function call in the AST
				callExpr := findCallExpression(t, program)
				if callExpr != nil {
					tt.checkArgs(t, callExpr)
				}
			}
		})
	}
}

func TestNamedArgumentErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
		description string
	}{
		{
			name: "Duplicate named argument",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(NAME, ";", name_case = LOWER, name_case = UPPER);
}`,
			expectedErr: "already used",
			description: "Same named argument used twice should error",
		},
		{
			name: "Malformed named argument - missing value",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(NAME, ";", name_case = );
}`,
			expectedErr: "",
			description: "Named argument without value should error",
		},
		{
			name: "Malformed named argument - missing equals",
			input: `vcl 4.0;
sub test {
	headerplus.as_list(NAME, ";", name_case LOWER);
}`,
			expectedErr: "",
			description: "Named argument without equals should error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			hasErrors := len(p.Errors()) > 0
			if !hasErrors {
				t.Errorf("Test case: %s", tt.description)
				t.Errorf("Expected parse error but got none")
				return
			}

			// Check for specific error message if provided
			if tt.expectedErr != "" {
				found := false
				for _, err := range p.Errors() {
					if contains(err.Error(), tt.expectedErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s', got errors: %v", tt.expectedErr, p.Errors())
				}
			}

			_ = program // Avoid unused variable warning
		})
	}
}

// Helper function to find the first CallExpression in a program
func findCallExpression(t *testing.T, program *ast2.Program) *ast2.CallExpression {
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast2.SubDecl); ok {
			return findCallExpressionInBlock(subDecl.Body)
		}
	}
	t.Error("No subroutine declaration found in program")
	return nil
}

// Helper function to find CallExpression in a block statement
func findCallExpressionInBlock(block *ast2.BlockStatement) *ast2.CallExpression {
	for _, stmt := range block.Statements {
		if setStmt, ok := stmt.(*ast2.SetStatement); ok {
			return findCallExpressionInExpression(setStmt.Value)
		}
		if exprStmt, ok := stmt.(*ast2.ExpressionStatement); ok {
			return findCallExpressionInExpression(exprStmt.Expression)
		}
	}
	return nil
}

// Helper function to find CallExpression in an expression
func findCallExpressionInExpression(expr ast2.Expression) *ast2.CallExpression {
	if callExpr, ok := expr.(*ast2.CallExpression); ok {
		return callExpr
	}
	// Could recursively search in other expression types if needed
	return nil
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsInMiddle(s, substr))))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
