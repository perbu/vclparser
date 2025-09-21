package vclparser_test

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

func TestCallStatementParsing(t *testing.T) {
	vclCode := `vcl 4.0;

sub test_sub {
    set req.http.X-Test = "value";
}

sub vcl_recv {
    call test_sub;
    return (pass);
}`

	program, err := parser.Parse(vclCode, "test.vcl")
	if err != nil {
		if strings.Contains(err.Error(), "call") || strings.Contains(err.Error(), "unexpected token") {
			t.Fatal("Call statement parsing not yet implemented")
		}
		t.Fatalf("Unexpected parsing error: %v", err)
	}

	// If we get here, call statements work!
	t.Log("Basic call statement parsing is working!")

	// Verify we have the expected subroutines
	counts := countDeclarationsByType(program)
	if counts["subroutine"] != 2 {
		t.Errorf("Expected 2 subroutines, got %d", counts["subroutine"])
	}
}

func TestCallStatementParsingInvalid(t *testing.T) {
	vclCode := `vcl 4.0;

sub vcl_recv {
    call test_sub;
    return (pass);
}`

	if _, err := parser.Parse(vclCode, "test.vcl"); err == nil {
		t.Fatal("Expected parsing error, got none")
	}
}
