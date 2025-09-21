package parser

import (
	"testing"

	ast2 "github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
)

func TestReturnActions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		action string
	}{
		// Return actions with parentheses (required syntax)
		{"parenthesized lookup", `vcl 4.1; sub test { return (lookup); }`, "lookup"},
		{"parenthesized hash", `vcl 4.1; sub test { return (hash); }`, "hash"},
		{"parenthesized pass", `vcl 4.1; sub test { return (pass); }`, "pass"},
		{"parenthesized pipe", `vcl 4.1; sub test { return (pipe); }`, "pipe"},
		{"parenthesized purge", `vcl 4.1; sub test { return (purge); }`, "purge"},
		{"parenthesized deliver", `vcl 4.1; sub test { return (deliver); }`, "deliver"},
		{"parenthesized restart", `vcl 4.1; sub test { return (restart); }`, "restart"},
		{"parenthesized fetch", `vcl 4.1; sub test { return (fetch); }`, "fetch"},
		{"parenthesized miss", `vcl 4.1; sub test { return (miss); }`, "miss"},
		{"parenthesized hit", `vcl 4.1; sub test { return (hit); }`, "hit"},
		{"parenthesized synth", `vcl 4.1; sub test { return (synth); }`, "synth"},
		{"parenthesized abandon", `vcl 4.1; sub test { return (abandon); }`, "abandon"},
		{"parenthesized retry", `vcl 4.1; sub test { return (retry); }`, "retry"},
		{"parenthesized error", `vcl 4.1; sub test { return (error); }`, "error"},
		{"parenthesized ok", `vcl 4.1; sub test { return (ok); }`, "ok"},
		{"parenthesized fail", `vcl 4.1; sub test { return (fail); }`, "fail"},
		{"parenthesized vcl", `vcl 4.1; sub test { return (vcl); }`, "vcl"},
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
			var sub *ast2.SubDecl
			for _, decl := range program.Declarations {
				if s, ok := decl.(*ast2.SubDecl); ok && s.Name == "test" {
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

			retStmt, ok := sub.Body.Statements[0].(*ast2.ReturnStatement)
			if !ok {
				t.Fatalf("first statement is not a return statement, got %T", sub.Body.Statements[0])
			}

			if retStmt.Action == nil {
				t.Fatalf("return statement has no action")
			}

			// Check the action is an identifier with the expected value
			ident, ok := retStmt.Action.(*ast2.Identifier)
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
		{"synth with args", `vcl 4.1; sub test { return (synth(404, "Not Found")); }`},
		{"error with args", `vcl 4.1; sub test { return (error(500)); }`},
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
			var sub *ast2.SubDecl
			for _, decl := range program.Declarations {
				if s, ok := decl.(*ast2.SubDecl); ok && s.Name == "test" {
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

			retStmt, ok := sub.Body.Statements[0].(*ast2.ReturnStatement)
			if !ok {
				t.Fatalf("first statement is not a return statement, got %T", sub.Body.Statements[0])
			}

			if retStmt.Action == nil {
				t.Fatalf("return statement has no action")
			}

			// For function calls, we expect a CallExpression
			if _, ok := retStmt.Action.(*ast2.CallExpression); !ok {
				// Could also be a ParenthesizedExpression containing a CallExpression
				if parenExpr, isParenExpr := retStmt.Action.(*ast2.ParenthesizedExpression); isParenExpr {
					if _, ok := parenExpr.Expression.(*ast2.CallExpression); !ok {
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
	tests := []struct {
		name  string
		input string
	}{
		{"naked return rejected", `vcl 4.1; sub test { return lookup; }`},
		{"naked return with semicolon rejected", `vcl 4.1; sub test { return hash; }`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.input, "test.vcl")
			p := New(l, tt.input, "test.vcl")
			program := p.ParseProgram()

			if program == nil {
				t.Fatalf("program is nil")
			}

			// Find the subroutine
			var sub *ast2.SubDecl
			for _, decl := range program.Declarations {
				if s, ok := decl.(*ast2.SubDecl); ok && s.Name == "test" {
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

			retStmt, ok := sub.Body.Statements[0].(*ast2.ReturnStatement)
			if !ok {
				t.Fatalf("first statement is not a return statement, got %T", sub.Body.Statements[0])
			}

			// With naked returns removed, the action should be nil
			if retStmt.Action != nil {
				t.Errorf("expected return statement to have no action, but got %T", retStmt.Action)
			}
		})
	}
}
