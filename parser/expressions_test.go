package parser

import (
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

func TestCallExpressions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // Expected function name
		argCount int
		wantErr  bool
	}{
		{
			name:     "simple function call no args",
			input:    `vcl 4.0; sub test { func(); }`,
			expected: "func",
			argCount: 0,
			wantErr:  false,
		},
		{
			name:     "function call with one arg",
			input:    `vcl 4.0; sub test { func(arg); }`,
			expected: "func",
			argCount: 1,
			wantErr:  false,
		},
		{
			name:     "function call with multiple args",
			input:    `vcl 4.0; sub test { func(arg1, arg2, arg3); }`,
			expected: "func",
			argCount: 3,
			wantErr:  false,
		},
		{
			name:     "member function call no args",
			input:    `vcl 4.0; sub test { cluster.backend(); }`,
			expected: "cluster", // Function is the member expression
			argCount: 0,
			wantErr:  false,
		},
		{
			name:     "member function call with args",
			input:    `vcl 4.0; sub test { cluster.add_backend(web1); }`,
			expected: "cluster",
			argCount: 1,
			wantErr:  false,
		},
		{
			name:     "nested function calls",
			input:    `vcl 4.0; sub test { outer(inner()); }`,
			expected: "outer",
			argCount: 1,
			wantErr:  false, // Nested calls are now supported!
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			if tt.wantErr {
				if len(p.Errors()) == 0 {
					t.Error("Expected parser errors, but got none")
				}
				return
			}

			if len(p.Errors()) > 0 {
				t.Errorf("Unexpected parser errors: %v", p.Errors())
				return
			}

			// Navigate to the call expression
			subDecl := program.Declarations[0].(*ast.SubDecl)
			exprStmt := subDecl.Body.Statements[0].(*ast.ExpressionStatement)
			callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
			if !ok {
				t.Fatalf("Expected CallExpression, got %T", exprStmt.Expression)
			}

			// Check argument count
			if len(callExpr.Arguments) != tt.argCount {
				t.Errorf("Expected %d arguments, got %d", tt.argCount, len(callExpr.Arguments))
			}

			// Verify no nil arguments
			for i, arg := range callExpr.Arguments {
				if arg == nil {
					t.Errorf("Argument %d is nil", i)
				}
			}

			// Check that End() doesn't panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("CallExpression.End() panicked: %v", r)
					}
				}()
				_ = callExpr.End()
			}()

			// Verify function name for simple cases
			if tt.expected != "" {
				switch fn := callExpr.Function.(type) {
				case *ast.Identifier:
					if fn.Name != tt.expected {
						t.Errorf("Expected function name %s, got %s", tt.expected, fn.Name)
					}
				case *ast.MemberExpression:
					if obj, ok := fn.Object.(*ast.Identifier); ok && obj.Name != tt.expected {
						t.Errorf("Expected object name %s, got %s", tt.expected, obj.Name)
					}
				}
			}
		})
	}
}

func TestCallExpressionPanicRegression(t *testing.T) {
	// Specific test for the panic issue reported
	input := `vcl 4.0;
sub test {
    cluster.backend();
}`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	checkParserErrors(t, p)

	// Navigate to the call expression
	subDecl := program.Declarations[0].(*ast.SubDecl)
	exprStmt := subDecl.Body.Statements[0].(*ast.ExpressionStatement)
	callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpression, got %T", exprStmt.Expression)
	}

	// This should not panic
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("CallExpression.End() panicked: %v", r)
			}
		}()
		_ = callExpr.End()
		_ = exprStmt.End()
	}()

	// Verify the structure
	memberExpr, ok := callExpr.Function.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpression as function, got %T", callExpr.Function)
	}

	if obj, ok := memberExpr.Object.(*ast.Identifier); !ok || obj.Name != "cluster" {
		t.Errorf("Expected object 'cluster', got %v", memberExpr.Object)
	}

	if prop, ok := memberExpr.Property.(*ast.Identifier); !ok || prop.Name != "backend" {
		t.Errorf("Expected property 'backend', got %v", memberExpr.Property)
	}

	if len(callExpr.Arguments) != 0 {
		t.Errorf("Expected 0 arguments, got %d", len(callExpr.Arguments))
	}
}

func TestCallExpressionWithInvalidArguments(t *testing.T) {
	// Test case that should produce errors
	input := `vcl 4.0;
sub test {
    func(,);
}`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	// This should produce errors due to invalid syntax
	if len(p.Errors()) == 0 {
		t.Error("Expected parser errors for invalid arguments, but got none")
	}

	// Check that we don't have a call expression (parsing should fail)
	if len(program.Declarations) > 0 {
		if subDecl, ok := program.Declarations[0].(*ast.SubDecl); ok {
			if len(subDecl.Body.Statements) > 0 {
				if exprStmt, ok := subDecl.Body.Statements[0].(*ast.ExpressionStatement); ok {
					if exprStmt.Expression == nil {
						// This is expected - the expression statement should be nil due to parsing failure
						return
					}
				}
			}
		}
	}
}

func TestExpressionStatementEnd(t *testing.T) {
	// Test that expression statements can call End() without panicking
	tests := []string{
		`vcl 4.0; sub test { req.method; }`,
		`vcl 4.0; sub test { func(); }`,
		`vcl 4.0; sub test { obj.method(); }`,
		`vcl 4.0; sub test { func(arg1, arg2); }`,
	}

	for i, input := range tests {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			l := lexer.New(input, "test.vcl")
			p := New(l, input, "test.vcl")
			program := p.ParseProgram()

			checkParserErrors(t, p)

			subDecl := program.Declarations[0].(*ast.SubDecl)
			exprStmt := subDecl.Body.Statements[0].(*ast.ExpressionStatement)

			// This should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("ExpressionStatement.End() panicked: %v", r)
					}
				}()
				_ = exprStmt.End()
			}()
		})
	}
}
