package vclparser_test

import (
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

func TestNamedParametersWithIdentifiers(t *testing.T) {
	vcl := `vcl 4.0;

import directors;

backend default_backend {
	.host = "localhost";
	.port = "8080";
}

probe default_error_probe {
	.url = "/health";
}

sub vcl_init {
	new director1 = directors.round_robin();
	new director2 = directors.random(probe=default_error_probe);
}
`

	program, err := parser.Parse(vcl, "test.vcl")
	if err != nil {
		t.Fatalf("Expected no errors, got: %v", err)
	}
	if program == nil {
		t.Fatal("Expected valid program")
	}
}
