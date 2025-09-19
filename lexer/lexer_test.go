package lexer

import (
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `vcl 4.0;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    if (req.method == "GET") {
        return (hash);
    }
}`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{VCL_KW, "vcl"},
		{FNUM, "4.0"},
		{SEMICOLON, ";"},
		{BACKEND_KW, "backend"},
		{ID, "default"},
		{LBRACE, "{"},
		{DOT, "."},
		{ID, "host"},
		{ASSIGN, "="},
		{CSTR, `"127.0.0.1"`},
		{SEMICOLON, ";"},
		{DOT, "."},
		{ID, "port"},
		{ASSIGN, "="},
		{CSTR, `"8080"`},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{SUB_KW, "sub"},
		{ID, "vcl_recv"},
		{LBRACE, "{"},
		{IF_KW, "if"},
		{LPAREN, "("},
		{ID, "req"},
		{DOT, "."},
		{ID, "method"},
		{EQ, "=="},
		{CSTR, `"GET"`},
		{RPAREN, ")"},
		{LBRACE, "{"},
		{RETURN_KW, "return"},
		{LPAREN, "("},
		{HASH_KW, "hash"},
		{RPAREN, ")"},
		{SEMICOLON, ";"},
		{RBRACE, "}"},
		{RBRACE, "}"},
		{EOF, ""},
	}

	l := New(input, "test.vcl")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Value != tt.expectedValue {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Value)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `== != <= >= && || += -= *= /= ++ -- << >> !~`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{EQ, "=="},
		{NEQ, "!="},
		{LEQ, "<="},
		{GEQ, ">="},
		{CAND, "&&"},
		{COR, "||"},
		{INCR, "+="},
		{DECR, "-="},
		{MUL, "*="},
		{DIV, "/="},
		{INC, "++"},
		{DEC, "--"},
		{SHL, "<<"},
		{SHR, ">>"},
		{NOMATCH, "!~"},
		{EOF, ""},
	}

	l := New(input, "test.vcl")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Value != tt.expectedValue {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Value)
		}
	}
}

func TestComments(t *testing.T) {
	input := `// Single line comment
/* Multi-line
   comment */
# Shell style comment
vcl 4.0;`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{COMMENT, "// Single line comment"},
		{COMMENT, "/* Multi-line\n   comment */"},
		{COMMENT, "# Shell style comment"},
		{VCL_KW, "vcl"},
		{FNUM, "4.0"},
		{SEMICOLON, ";"},
		{EOF, ""},
	}

	l := New(input, "test.vcl")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Value != tt.expectedValue {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Value)
		}
	}
}

func TestCBlock(t *testing.T) {
	input := `C{
    #include <stdio.h>
    printf("Hello from C\n");
}C`

	expected := `C{
    #include <stdio.h>
    printf("Hello from C\n");
}C`

	l := New(input, "test.vcl")
	tok := l.NextToken()

	if tok.Type != CSRC {
		t.Fatalf("expected CSRC token, got %q", tok.Type)
	}

	if tok.Value != expected {
		t.Fatalf("expected %q, got %q", expected, tok.Value)
	}
}

func TestNumbers(t *testing.T) {
	input := `123 456.789 3.14e10 2E-5`

	tests := []struct {
		expectedType  TokenType
		expectedValue string
	}{
		{CNUM, "123"},
		{FNUM, "456.789"},
		{FNUM, "3.14e10"},
		{FNUM, "2E-5"},
		{EOF, ""},
	}

	l := New(input, "test.vcl")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Value != tt.expectedValue {
			t.Fatalf("test[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedValue, tok.Value)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `vcl backend sub probe acl import include if else set unset call return`

	tests := []struct {
		expectedType TokenType
	}{
		{VCL_KW},
		{BACKEND_KW},
		{SUB_KW},
		{PROBE_KW},
		{ACL_KW},
		{IMPORT_KW},
		{INCLUDE_KW},
		{IF_KW},
		{ELSE_KW},
		{SET_KW},
		{UNSET_KW},
		{CALL_KW},
		{RETURN_KW},
		{EOF},
	}

	l := New(input, "test.vcl")

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("test[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}
	}
}
