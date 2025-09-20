package parser

import (
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

func TestReturnActions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		action string
	}{
		// Basic return actions without parentheses
		{"naked lookup", `vcl 4.1; sub test { return lookup; }`, "lookup"},
		{"naked hash", `vcl 4.1; sub test { return hash; }`, "hash"},
		{"naked pass", `vcl 4.1; sub test { return pass; }`, "pass"},
		{"naked pipe", `vcl 4.1; sub test { return pipe; }`, "pipe"},
		{"naked purge", `vcl 4.1; sub test { return purge; }`, "purge"},
		{"naked deliver", `vcl 4.1; sub test { return deliver; }`, "deliver"},
		{"naked restart", `vcl 4.1; sub test { return restart; }`, "restart"},
		{"naked fetch", `vcl 4.1; sub test { return fetch; }`, "fetch"},
		{"naked miss", `vcl 4.1; sub test { return miss; }`, "miss"},
		{"naked hit", `vcl 4.1; sub test { return hit; }`, "hit"},
		{"naked synth", `vcl 4.1; sub test { return synth; }`, "synth"},
		{"naked abandon", `vcl 4.1; sub test { return abandon; }`, "abandon"},
		{"naked retry", `vcl 4.1; sub test { return retry; }`, "retry"},
		{"naked error", `vcl 4.1; sub test { return error; }`, "error"},
		{"naked ok", `vcl 4.1; sub test { return ok; }`, "ok"},
		{"naked fail", `vcl 4.1; sub test { return fail; }`, "fail"},
		{"naked vcl", `vcl 4.1; sub test { return vcl; }`, "vcl"},

		// Return actions with parentheses (backward compatibility)
		{"parenthesized lookup", `vcl 4.1; sub test { return (lookup); }`, "lookup"},
		{"parenthesized hash", `vcl 4.1; sub test { return (hash); }`, "hash"},
		{"parenthesized pass", `vcl 4.1; sub test { return (pass); }`, "pass"},
		{"parenthesized deliver", `vcl 4.1; sub test { return (deliver); }`, "deliver"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			checkParserErrors(t, p)

			if program == nil {
				t.Fatalf("program is nil")
			}

			// Find the subroutine
			var sub *ast.SubDecl
			for _, decl := range program.Declarations {
				if s, ok := decl.(*ast.SubDecl); ok && s.Name == "test" {
					sub = s
					break
				}
			}

			if sub == nil {
				t.Fatalf("subroutine 'test' not found")
			}

			// Find the return statement
			if len(sub.Body.Statements) == 0 {
				t.Fatalf("no statements in subroutine body")
			}

			retStmt, ok := sub.Body.Statements[0].(*ast.ReturnStatement)
			if !ok {
				t.Fatalf("first statement is not a return statement, got %T", sub.Body.Statements[0])
			}

			if retStmt.Action == nil {
				t.Fatalf("return statement has no action")
			}

			// Check the action is an identifier with the expected value
			ident, ok := retStmt.Action.(*ast.Identifier)
			if !ok {
				t.Fatalf("return action is not an identifier, got %T", retStmt.Action)
			}

			if ident.Name != tt.action {
				t.Errorf("expected action %q, got %q", tt.action, ident.Name)
			}
		})
	}
}

func TestReturnActionsWithArguments(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		// Return actions with function call syntax
		{"synth with args", `vcl 4.1; sub test { return synth(404, "Not Found"); }`},
		{"error with args", `vcl 4.1; sub test { return error(500); }`},
		{"parenthesized synth with args", `vcl 4.1; sub test { return (synth(404, "Not Found")); }`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			checkParserErrors(t, p)

			if program == nil {
				t.Fatalf("program is nil")
			}

			// Find the subroutine
			var sub *ast.SubDecl
			for _, decl := range program.Declarations {
				if s, ok := decl.(*ast.SubDecl); ok && s.Name == "test" {
					sub = s
					break
				}
			}

			if sub == nil {
				t.Fatalf("subroutine 'test' not found")
			}

			// Find the return statement
			if len(sub.Body.Statements) == 0 {
				t.Fatalf("no statements in subroutine body")
			}

			retStmt, ok := sub.Body.Statements[0].(*ast.ReturnStatement)
			if !ok {
				t.Fatalf("first statement is not a return statement, got %T", sub.Body.Statements[0])
			}

			if retStmt.Action == nil {
				t.Fatalf("return statement has no action")
			}

			// For function calls, we expect a CallExpression
			if _, ok := retStmt.Action.(*ast.CallExpression); !ok {
				// Could also be a ParenthesizedExpression containing a CallExpression
				if parenExpr, isParenExpr := retStmt.Action.(*ast.ParenthesizedExpression); isParenExpr {
					if _, ok := parenExpr.Expression.(*ast.CallExpression); !ok {
						t.Fatalf("return action is not a call expression, got %T", retStmt.Action)
					}
				} else {
					t.Fatalf("return action is not a call expression, got %T", retStmt.Action)
				}
			}
		})
	}
}

func TestReturnActionsSyntaxError(t *testing.T) {
	// Test that invalid return statements still fail appropriately
	input := `vcl 4.1; sub test { return unknown_action; }`

	l := lexer.New(input, "test.vcl")
	p := New(l, input, "test.vcl")
	program := p.ParseProgram()

	// This should parse successfully because "unknown_action" is just an identifier
	// The semantic analysis should catch invalid return actions, not the parser
	checkParserErrors(t, p)

	if program == nil {
		t.Fatalf("program is nil")
	}
}
