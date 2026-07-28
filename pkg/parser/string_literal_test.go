package parser

import (
	"testing"

	ast2 "github.com/perbu/vclparser/pkg/ast"
)

// StringLiteral.Value holds the raw bytes between the delimiters. VCL has no
// escape sequences inside strings, so the parser must not decode anything and
// the renderer must not re-encode anything.

func TestStringLiteralValueIsRaw(t *testing.T) {
	tests := []struct {
		name     string
		stmt     string
		want     string
		wantLong bool
	}{
		{
			name: "backslashes survive unchanged",
			stmt: `set req.http.X = "\.jpg$";`,
			want: `\.jpg$`,
		},
		{
			name: "escaped parens survive unchanged",
			stmt: `set req.http.X = "\(compatible\)";`,
			want: `\(compatible\)`,
		},
		{
			name: "empty string is empty, not mangled by trimming",
			stmt: `set req.http.X = "";`,
			want: ``,
		},
		{
			name: "value ending in a backslash",
			stmt: `set req.http.X = "c:\\";`,
			want: `c:\\`,
		},
		{
			name:     "long string records its form",
			stmt:     `set req.http.X = {"hello"};`,
			want:     `hello`,
			wantLong: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "vcl 4.1;\n\nsub vcl_recv {\n    " + tt.stmt + "\n}\n"
			program, err := Parse(src, "test.vcl")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			lit := firstStringLiteral(t, program)
			if lit.Value != tt.want {
				t.Errorf("Value = %q, want %q", lit.Value, tt.want)
			}
			if lit.Long != tt.wantLong {
				t.Errorf("Long = %v, want %v", lit.Long, tt.wantLong)
			}
		})
	}
}

// firstStringLiteral digs the single string literal out of the first set
// statement of the first subroutine.
func firstStringLiteral(t *testing.T, program *ast2.Program) *ast2.StringLiteral {
	t.Helper()

	for _, decl := range program.Declarations {
		sub, ok := decl.(*ast2.SubDecl)
		if !ok || sub.Body == nil {
			continue
		}
		for _, stmt := range sub.Body.Statements {
			set, ok := stmt.(*ast2.SetStatement)
			if !ok {
				continue
			}
			if lit, ok := set.Value.(*ast2.StringLiteral); ok {
				return lit
			}
		}
	}

	t.Fatal("no string literal found in the parsed program")
	return nil
}
