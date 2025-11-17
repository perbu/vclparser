package vclparser_test

import (
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

func TestHTTPHeaderWithHyphens(t *testing.T) {
	tests := []struct {
		name   string
		vcl    string
		errMsg string
	}{
		{
			name: "simple header with single hyphen",
			vcl: `vcl 4.0;
sub vcl_recv {
	if (req.http.X-Forwarded-For) {
		return (pass);
	}
}`,
		},
		{
			name: "header with multiple hyphens",
			vcl: `vcl 4.0;
sub vcl_recv {
	if (req.http.x-default-backend-selected) {
		return (pass);
	}
}`,
		},
		{
			name: "header assignment",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.X-Custom-Header = "value";
}`,
		},
		{
			name: "complex expression with hyphenated headers",
			vcl: `vcl 4.0;
sub vcl_recv {
	if (!req.http.X-amedia-app && !req.http.x-default-backend-selected && req.http.x-domain-default-backend) {
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
