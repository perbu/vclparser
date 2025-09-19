package parser

import (
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

func TestNestedFunctionCalls(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		description string
	}{
		// Examples from nested-calls.md document
		{
			name: "Example 1: Function call as function argument (from doc line 27)",
			input: `vcl 4.0;
sub test {
	regsub(req.url, "counter", kvs.counter("c", 1));
}`,
			wantErr:     false,
			description: "kvs.counter() result passed as third argument to regsub()",
		},
		{
			name: "Example 2: Named parameter with nested call (from doc line 36)",
			input: `vcl 4.0;
sub test {
	xbody.regsub("\Steven", "Andrew", max = std.integer(bereq.http.max, 0));
}`,
			wantErr:     true, // Named parameters not yet supported
			description: "std.integer() used in named parameter - LIMITATION: named params not supported",
		},
		{
			name: "Simple nested call - function as argument",
			input: `vcl 4.0;
sub test {
	std.tolower(std.toupper("test"));
}`,
			wantErr:     false,
			description: "std.toupper() result passed to std.tolower()",
		},
		{
			name: "String manipulation chain (from doc line 92)",
			input: `vcl 4.0;
sub test {
	regsub(
		regsub(req.url, "^/old/", "/new/"),
		"\.html$",
		".php"
	);
}`,
			wantErr:     false, // Multi-line function calls actually work!
			description: "Nested regsub calls for chained string manipulation",
		},
		{
			name: "Type conversion with nested call (from doc line 100)",
			input: `vcl 4.0;
sub test {
	std.integer(
		regsub(req.http.counter, "[^0-9]", ""),
		0
	);
}`,
			wantErr:     false, // Multi-line function calls actually work!
			description: "regsub() result passed to std.integer()",
		},
		{
			name: "Simple function call with literal argument",
			input: `vcl 4.0;
sub test {
	std.tolower("TEST");
}`,
			wantErr:     false,
			description: "Baseline test - single function call with literal",
		},
		{
			name: "Function call with variable argument",
			input: `vcl 4.0;
sub test {
	std.tolower(req.url);
}`,
			wantErr:     false,
			description: "Baseline test - single function call with variable",
		},
		{
			name: "Multiple arguments including nested call",
			input: `vcl 4.0;
sub test {
	regsub(req.url, "pattern", std.tolower("REPLACEMENT"));
}`,
			wantErr:     false,
			description: "Function call as one of multiple arguments",
		},
		{
			name: "Deep nesting - three levels",
			input: `vcl 4.0;
sub test {
	std.tolower(regsub(std.toupper(req.url), "A", "B"));
}`,
			wantErr:     false,
			description: "Three levels of nested function calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l)
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
			}

			// If parsing succeeded and we expected success, verify the AST structure
			if !hasErrors && !tt.wantErr && program != nil {
				// Check that we have at least one declaration (the subroutine)
				if len(program.Declarations) == 0 {
					t.Errorf("No declarations found in parsed program")
				} else {
					// Verify it's a subroutine
					if sub, ok := program.Declarations[0].(*ast.SubDecl); ok {
						if len(sub.Body.Statements) == 0 {
							t.Errorf("Subroutine body is empty")
						}
					} else {
						t.Errorf("First declaration is not a subroutine, got %T", program.Declarations[0])
					}
				}
			}
		})
	}
}

// TestNestedCallsInExpressions tests nested calls in various expression contexts
func TestNestedCallsInExpressions(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "Nested call in if condition",
			input: `vcl 4.0;
sub test {
	if (std.integer(req.http.count, 0) > 5) {
		return (pass);
	}
}`,
			wantErr: false,
		},
		{
			name: "Nested call in binary expression",
			input: `vcl 4.0;
sub test {
	if (req.url ~ std.tolower("PATTERN")) {
		return (pass);
	}
}`,
			wantErr: false,
		},
		{
			name: "Nested call with member access",
			input: `vcl 4.0;
sub test {
	std.tolower(req.http.path);
}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l)
			_ = p.ParseProgram()

			hasErrors := len(p.Errors()) > 0

			if hasErrors != tt.wantErr {
				if tt.wantErr {
					t.Errorf("Expected parse error but got none")
				} else {
					t.Errorf("Unexpected parse errors:")
					for _, err := range p.Errors() {
						t.Errorf("  - %s", err)
					}
				}
			}
		})
	}
}

// TestCallExpressionParsing specifically tests the parseCallExpression function
func TestCallExpressionParsing(t *testing.T) {
	// Test that parseCallExpression correctly handles recursive parsing
	input := `vcl 4.0;
sub test {
	foo(bar("test"));
}`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Errorf("Parse errors encountered:")
		for _, err := range p.Errors() {
			t.Errorf("  - %s", err)
		}
		return
	}

	// Navigate to the call expression in the AST
	if len(program.Declarations) == 0 {
		t.Fatal("No declarations found")
	}

	sub, ok := program.Declarations[0].(*ast.SubDecl)
	if !ok {
		t.Fatalf("Expected SubDecl, got %T", program.Declarations[0])
	}

	if len(sub.Body.Statements) == 0 {
		t.Fatal("Subroutine body is empty")
	}

	// The first statement should be an expression statement containing the call
	exprStmt, ok := sub.Body.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("Expected ExpressionStatement, got %T", sub.Body.Statements[0])
	}

	// Check if it's a call expression
	callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpression, got %T", exprStmt.Expression)
	}

	// Verify the outer function name
	if fn, ok := callExpr.Function.(*ast.Identifier); ok {
		if fn.Name != "foo" {
			t.Errorf("Expected function name 'foo', got '%s'", fn.Name)
		}
	}

	// Check that the argument is another call expression
	if len(callExpr.Arguments) != 1 {
		t.Fatalf("Expected 1 argument, got %d", len(callExpr.Arguments))
	}

	nestedCall, ok := callExpr.Arguments[0].(*ast.CallExpression)
	if !ok {
		t.Errorf("Expected nested CallExpression as argument, got %T", callExpr.Arguments[0])
	} else {
		// Verify the nested function name
		if fn, ok := nestedCall.Function.(*ast.Identifier); ok {
			if fn.Name != "bar" {
				t.Errorf("Expected nested function name 'bar', got '%s'", fn.Name)
			}
		}

		// Verify the nested function has one string argument
		if len(nestedCall.Arguments) != 1 {
			t.Errorf("Expected nested call to have 1 argument, got %d", len(nestedCall.Arguments))
		}
	}
}
