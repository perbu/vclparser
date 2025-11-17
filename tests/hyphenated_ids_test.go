package vclparser_test

import (
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

func TestHyphenatedIdentifiers(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "backend with hyphens",
			vcl: `vcl 4.0;
backend my-backend {
	.host = "localhost";
}`,
		},
		{
			name: "probe with hyphens",
			vcl: `vcl 4.0;
probe test-probe {
	.url = "/health";
}`,
		},
		{
			name: "HTTP header with hyphens",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.x-emergency-mode = "test";
}`,
		},
		{
			name: "synthetic with hyphenated header",
			vcl: `vcl 4.0;
sub vcl_synth {
	synthetic(req.http.x-emergency-mode);
}`,
		},
		{
			name: "variable with hyphens in expression",
			vcl: `vcl 4.0;
sub vcl_recv {
	if (req.http.x-test-header) {
		return (pass);
	}
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("Expected no errors, got: %v", err)
			}
			if program == nil {
				t.Fatal("Expected valid program")
			}
		})
	}
}
