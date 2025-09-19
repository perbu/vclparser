package parser

import (
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

func TestVCLVersionDeclaration(t *testing.T) {
	input := `vcl 4.0;`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if program.VCLVersion == nil {
		t.Fatal("program.VCLVersion is nil")
	}

	if program.VCLVersion.Version != "4.0" {
		t.Errorf("program.VCLVersion.Version = %q, want %q", program.VCLVersion.Version, "4.0")
	}
}

func TestBackendDeclaration(t *testing.T) {
	input := `vcl 4.0;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.BackendDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.BackendDecl. got=%T",
			program.Declarations[0])
	}

	if decl.Name != "default" {
		t.Errorf("decl.Name = %q, want %q", decl.Name, "default")
	}

	if len(decl.Properties) != 2 {
		t.Fatalf("backend does not contain 2 properties. got=%d", len(decl.Properties))
	}

	expectedProperties := []struct {
		name  string
		value string
	}{
		{"host", "127.0.0.1"},
		{"port", "8080"},
	}

	for i, expected := range expectedProperties {
		prop := decl.Properties[i]
		if prop.Name != expected.name {
			t.Errorf("property[%d].Name = %q, want %q", i, prop.Name, expected.name)
		}

		stringLit, ok := prop.Value.(*ast.StringLiteral)
		if !ok {
			t.Fatalf("property[%d].Value is not *ast.StringLiteral. got=%T", i, prop.Value)
		}

		if stringLit.Value != expected.value {
			t.Errorf("property[%d].Value = %q, want %q", i, stringLit.Value, expected.value)
		}
	}
}

func TestSubroutineDeclaration(t *testing.T) {
	input := `vcl 4.0;

sub vcl_recv {
    if (req.method == "GET") {
        return (hash);
    }
}`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.SubDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.SubDecl. got=%T",
			program.Declarations[0])
	}

	if decl.Name != "vcl_recv" {
		t.Errorf("decl.Name = %q, want %q", decl.Name, "vcl_recv")
	}

	if decl.Body == nil {
		t.Fatal("decl.Body is nil")
	}

	if len(decl.Body.Statements) != 1 {
		t.Fatalf("subroutine body does not contain 1 statement. got=%d",
			len(decl.Body.Statements))
	}

	ifStmt, ok := decl.Body.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("statement is not *ast.IfStatement. got=%T", decl.Body.Statements[0])
	}

	if ifStmt.Condition == nil {
		t.Fatal("ifStmt.Condition is nil")
	}

	if ifStmt.Then == nil {
		t.Fatal("ifStmt.Then is nil")
	}
}

func TestACLDeclaration(t *testing.T) {
	input := `vcl 4.0;

acl purge {
    "localhost";
    "127.0.0.1";
    !"192.168.1.100";
}`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
			len(program.Declarations))
	}

	decl, ok := program.Declarations[0].(*ast.ACLDecl)
	if !ok {
		t.Fatalf("program.Declarations[0] is not *ast.ACLDecl. got=%T",
			program.Declarations[0])
	}

	if decl.Name != "purge" {
		t.Errorf("decl.Name = %q, want %q", decl.Name, "purge")
	}

	if len(decl.Entries) != 3 {
		t.Fatalf("ACL does not contain 3 entries. got=%d", len(decl.Entries))
	}

	// Check negated entry
	if !decl.Entries[2].Negated {
		t.Error("third ACL entry should be negated")
	}
}

func TestExpressionInCondition(t *testing.T) {
	input := `vcl 4.0; sub test { if (req.method) { return (hash); } }`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	_ = p.ParseProgram()

	errors := p.Errors()
	if len(errors) > 0 {
		t.Errorf("unexpected errors:")
		for _, err := range errors {
			t.Errorf("  error: %s", err.Message)
		}
		t.FailNow()
	}

	t.Logf("Expression in condition parsed successfully")
}

func TestSimpleExpressions(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
	}{
		{"simple identifier", "req", false},
		{"member access", "req.method", false},
		{"string comparison", "req.method == \"GET\"", false},
		{"regex match", "client.ip ~ acl", false},
		{"numeric comparison", "obj.hits > 0", false},
		{"parenthesized expression", "(req.method == \"GET\")", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := "vcl 4.0; sub test { " + tt.input + "; }"

			l := lexer.New(input, "test.vcl")
			p := New(l)
			program := p.ParseProgram()

			errors := p.Errors()
			if tt.expectError && len(errors) == 0 {
				t.Fatalf("expected error but got none for input: %s", tt.input)
			}
			if !tt.expectError && len(errors) > 0 {
				t.Errorf("unexpected errors for input: %s", tt.input)
				for _, err := range errors {
					t.Errorf("  error: %s", err.Message)
				}
				t.FailNow()
			}

			if tt.expectError {
				return // Skip further checks if we expect an error
			}

			if len(program.Declarations) != 1 {
				t.Fatalf("program.Declarations does not contain 1 declaration. got=%d",
					len(program.Declarations))
			}

			sub, ok := program.Declarations[0].(*ast.SubDecl)
			if !ok {
				t.Fatalf("declaration is not *ast.SubDecl. got=%T", program.Declarations[0])
			}

			if len(sub.Body.Statements) != 1 {
				t.Fatalf("subroutine body does not contain 1 statement. got=%d",
					len(sub.Body.Statements))
			}

			exprStmt, ok := sub.Body.Statements[0].(*ast.ExpressionStatement)
			if !ok {
				t.Fatalf("statement is not *ast.ExpressionStatement. got=%T",
					sub.Body.Statements[0])
			}

			if exprStmt.Expression == nil {
				t.Fatal("expression is nil")
			}
		})
	}
}

// TestCallExpressionStatementPanicRegression tests the specific panic issue
// where CallExpression.End() would panic with nil pointer dereference
func TestCallExpressionStatementPanicRegression(t *testing.T) {
	// This was the original failing case from PANIC.md
	input := `vcl 4.0;
sub test {
    cluster.backend();
}`

	l := lexer.New(input, "test.vcl")
	p := New(l)
	program := p.ParseProgram()

	checkParserErrors(t, p)

	if len(program.Declarations) != 1 {
		t.Fatalf("Expected 1 declaration, got %d", len(program.Declarations))
	}

	subDecl, ok := program.Declarations[0].(*ast.SubDecl)
	if !ok {
		t.Fatalf("Expected SubDecl, got %T", program.Declarations[0])
	}

	if len(subDecl.Body.Statements) != 1 {
		t.Fatalf("Expected 1 statement, got %d", len(subDecl.Body.Statements))
	}

	exprStmt, ok := subDecl.Body.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("Expected ExpressionStatement, got %T", subDecl.Body.Statements[0])
	}

	callExpr, ok := exprStmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("Expected CallExpression, got %T", exprStmt.Expression)
	}

	// This should not panic - was the original bug
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("CallExpression.End() panicked: %v", r)
			}
		}()

		// These calls were causing the panic
		_ = callExpr.End()
		_ = exprStmt.End()
	}()

	// Verify the structure is correct
	memberExpr, ok := callExpr.Function.(*ast.MemberExpression)
	if !ok {
		t.Fatalf("Expected MemberExpression as function, got %T", callExpr.Function)
	}

	objIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok || objIdent.Name != "cluster" {
		t.Errorf("Expected object 'cluster', got %v", memberExpr.Object)
	}

	propIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok || propIdent.Name != "backend" {
		t.Errorf("Expected property 'backend', got %v", memberExpr.Property)
	}

	if len(callExpr.Arguments) != 0 {
		t.Errorf("Expected 0 arguments, got %d", len(callExpr.Arguments))
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, err := range errors {
		t.Errorf("parser error: %s", err.Message)
	}
	t.FailNow()
}
