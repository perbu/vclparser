package renderer

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

// TestRenderProgram tests basic program rendering
func TestRenderProgram(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "simple vcl version",
			vcl:  "vcl 4.0;",
		},
		{
			name: "vcl with backend",
			vcl: `vcl 4.1;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`,
		},
		{
			name: "vcl with subroutine",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (req.method == "GET") {
        return(hash);
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the VCL
			program, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Render it back
			rendered := Render(program)

			// Verify it's not empty
			if strings.TrimSpace(rendered) == "" {
				t.Fatalf("Rendered VCL is empty")
			}

			t.Logf("Rendered VCL:\n%s", rendered)
		})
	}
}

// TestRoundTrip tests that parse -> render -> parse produces equivalent ASTs
func TestRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "simple backend",
			vcl: `vcl 4.0;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`,
		},
		{
			name: "if statement",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (req.method == "GET") {
        return(hash);
    }
}`,
		},
		{
			name: "if-else statement",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (req.method == "GET") {
        return(hash);
    } else {
        return(pass);
    }
}`,
		},
		{
			name: "set statement",
			vcl: `vcl 4.0;

sub vcl_recv {
    set req.http.Host = "example.com";
}`,
		},
		{
			name: "binary expressions",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (req.method == "GET" && req.url ~ "^/api") {
        return(pass);
    }
}`,
		},
		{
			name: "member access",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (req.http.host) {
        return(hash);
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First parse
			program1, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("First parse error: %v", err)
			}

			// Render
			rendered := Render(program1)
			t.Logf("Rendered VCL:\n%s", rendered)

			// Second parse
			program2, err := parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Second parse error on rendered VCL: %v\nRendered VCL:\n%s", err, rendered)
			}

			// Compare ASTs (simplified comparison - just check structure)
			if !comparePrograms(program1, program2) {
				t.Errorf("ASTs don't match after round-trip")
				t.Logf("Original VCL:\n%s", tt.vcl)
				t.Logf("Rendered VCL:\n%s", rendered)
			}
		})
	}
}

// TestRenderTestdataFiles tests rendering of actual test files
func TestRenderTestdataFiles(t *testing.T) {
	testFiles := []string{
		"../../tests/testdata/simple.vcl",
		"../../tests/testdata/return_actions.vcl",
	}

	for _, file := range testFiles {
		t.Run(filepath.Base(file), func(t *testing.T) {
			// Check if file exists
			if _, err := os.Stat(file); os.IsNotExist(err) {
				t.Skipf("Test file not found: %s", file)
			}

			// Read the file
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatalf("Failed to read file: %v", err)
			}

			// Parse
			program, err := parser.Parse(string(content), file)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Render
			rendered := Render(program)
			t.Logf("Rendered VCL:\n%s", rendered)

			// Parse rendered version
			program2, err := parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Failed to parse rendered VCL: %v\nRendered:\n%s", err, rendered)
			}

			// Basic validation - compare structure
			if len(program.Declarations) != len(program2.Declarations) {
				t.Errorf("Declaration count mismatch: original=%d, rendered=%d",
					len(program.Declarations), len(program2.Declarations))
			}
		})
	}
}

// TestRenderStatements tests various statement types
func TestRenderStatements(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "call statement",
			vcl: `vcl 4.0;

sub foo {
}

sub vcl_recv {
    call foo;
}`,
		},
		{
			name: "unset statement",
			vcl: `vcl 4.0;

sub vcl_recv {
    unset req.http.Cookie;
}`,
		},
		{
			name: "return statement",
			vcl: `vcl 4.0;

sub vcl_recv {
    return(hash);
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			rendered := Render(program)
			t.Logf("Rendered:\n%s", rendered)

			// Verify it parses
			_, err = parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Failed to parse rendered VCL: %v", err)
			}
		})
	}
}

// TestRenderExpressions tests various expression types
func TestRenderExpressions(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "string literal",
			vcl: `vcl 4.0;

sub vcl_recv {
    set req.http.Host = "example.com";
}`,
		},
		{
			name: "integer literal",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (beresp.status == 200) {
        return(deliver);
    }
}`,
		},
		{
			name: "boolean literal",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (true) {
        return(hash);
    }
}`,
		},
		{
			name: "unary expression",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (!req.http.host) {
        return(pass);
    }
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			rendered := Render(program)
			t.Logf("Rendered:\n%s", rendered)

			// Verify it parses
			_, err = parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Failed to parse rendered VCL: %v", err)
			}
		})
	}
}

// TestRenderDeclarations tests various declaration types
func TestRenderDeclarations(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "import declaration",
			vcl: `vcl 4.0;

import std;`,
		},
		{
			name: "probe declaration",
			vcl: `vcl 4.0;

probe healthcheck {
    .timeout = 5s;
    .interval = 10s;
}`,
		},
		{
			name: "acl declaration",
			vcl: `vcl 4.0;

acl local {
    "localhost";
    "192.168.0.0"/16;
}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			program, err := parser.Parse(tt.vcl, "test.vcl")
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			rendered := Render(program)
			t.Logf("Rendered:\n%s", rendered)

			// Verify it parses
			_, err = parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Failed to parse rendered VCL: %v", err)
			}
		})
	}
}

// comparePrograms performs a simplified comparison of two AST programs
func comparePrograms(p1, p2 *ast.Program) bool {
	// Compare VCL versions
	if p1.VCLVersion != nil && p2.VCLVersion != nil {
		if p1.VCLVersion.Version != p2.VCLVersion.Version {
			return false
		}
	} else if (p1.VCLVersion == nil) != (p2.VCLVersion == nil) {
		return false
	}

	// Compare declaration count
	if len(p1.Declarations) != len(p2.Declarations) {
		return false
	}

	// Compare declaration types (simplified)
	for i := range p1.Declarations {
		if reflect.TypeOf(p1.Declarations[i]) != reflect.TypeOf(p2.Declarations[i]) {
			return false
		}
	}

	return true
}
