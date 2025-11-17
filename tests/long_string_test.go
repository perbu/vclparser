package vclparser_test

import (
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

func TestLongStringLiterals(t *testing.T) {
	tests := []struct {
		name        string
		vcl         string
		expectError bool
		description string
	}{
		{
			name: "simple long string",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.test = {"hello world"};
}`,
			expectError: false,
			description: "Basic long string with simple text",
		},
		{
			name: "long string with embedded quotes",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.test = {"He said \"hello\" to me"};
}`,
			expectError: false,
			description: "Long string containing double quotes without escaping",
		},
		{
			name: "long string with regex pattern",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.tag = regsuball(req.http.test, {".+?group="([^"]*)"|.*$"}, "\1");
}`,
			expectError: false,
			description: "Long string with complex regex pattern from real-world VCL",
		},
		{
			name: "long string with multiple lines",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.multiline = {"Line 1
Line 2
Line 3"};
}`,
			expectError: false,
			description: "Long string spanning multiple lines",
		},
		{
			name: "long string with special characters",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.special = {"Special: \n \t \r | & $ @ # %"};
}`,
			expectError: false,
			description: "Long string with various special characters",
		},
		{
			name: "empty long string",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.empty = {""};
}`,
			expectError: false,
			description: "Empty long string",
		},
		{
			name: "long string with braces inside",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.braces = {"Contains { and } braces"};
}`,
			expectError: false,
			description: "Long string containing curly braces",
		},
		{
			name: "long string in regsuball",
			vcl: `vcl 4.0;
sub vcl_backend_response {
	set beresp.http.Amedia-Cache-Tag = regsuball(beresp.http.cache-control, {".+?group="([^"]*)"|.*$"}, "\1 ");
}`,
			expectError: false,
			description: "Real-world example from Amedia VCL",
		},
		{
			name: "long string in synthetic",
			vcl: `vcl 4.0;
sub vcl_synth {
	synthetic({"<!DOCTYPE html>
<html>
<head><title>Error</title></head>
<body>An error occurred</body>
</html>"});
}`,
			expectError: false,
			description: "Long string with HTML in synthetic()",
		},
		{
			name: "multiple long strings in one statement",
			vcl: `vcl 4.0;
sub vcl_recv {
	if (req.url ~ {"^/api/"} && req.http.host ~ {"example\.com"}) {
		return (pass);
	}
}`,
			expectError: false,
			description: "Multiple long strings in a single statement",
		},
		{
			name: "unterminated long string",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.bad = {"unterminated;
}`,
			expectError: true,
			description: "Should error on unterminated long string",
		},
		{
			name: "long string with pipe character",
			vcl: `vcl 4.0;
sub vcl_recv {
	set req.http.pattern = {"alpha|beta|gamma"};
}`,
			expectError: false,
			description: "Long string with pipe character (regex alternation)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.vcl, "test.vcl")

			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for: %s, but got none", tt.description)
				}
				// Got expected error
				return
			}

			if err != nil {
				t.Fatalf("Expected no errors for: %s\nGot: %v", tt.description, err)
			}
			if program == nil {
				t.Fatalf("Expected valid program for: %s", tt.description)
			}
		})
	}
}
