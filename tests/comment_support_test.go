package vclparser

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
	"github.com/perbu/vclparser/pkg/renderer"
)

// TestCommentPreservation tests that comments are preserved during parsing
func TestCommentPreservation(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "line comment before declaration",
			vcl: `vcl 4.0;

// This is a backend server
backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`,
		},
		{
			name: "block comment before declaration",
			vcl: `vcl 4.0;

/* This is a backend server
   with multiple lines */
backend default {
    .host = "127.0.0.1";
}`,
		},
		{
			name: "shell-style comment",
			vcl: `vcl 4.0;

# This is a shell-style comment
backend default {
    .host = "127.0.0.1";
}`,
		},
		{
			name: "comment before subroutine",
			vcl: `vcl 4.0;

// Handle incoming requests
sub vcl_recv {
    // Check request method
    if (req.method == "GET") {
        return(hash);
    }
}`,
		},
		{
			name: "comments on statements",
			vcl: `vcl 4.0;

sub vcl_recv {
    // Set custom header
    set req.http.X-Custom = "value";

    // Return hash
    return(hash); // end of processing
}`,
		},
		{
			name: "multiple leading comments",
			vcl: `vcl 4.0;

// First comment
// Second comment
// Third comment
backend default {
    .host = "127.0.0.1";
}`,
		},
		{
			name: "comment on vcl version",
			vcl: `// VCL version declaration
vcl 4.0; // version 4.0

backend default {
    .host = "127.0.0.1";
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

			// Check that program was parsed
			if program == nil {
				t.Fatal("Program is nil")
			}

			// Render it back
			rendered := renderer.Render(program)
			t.Logf("Original VCL:\n%s", tt.vcl)
			t.Logf("Rendered VCL:\n%s", rendered)

			// Verify comments are in the rendered output
			// This is a basic check - we'll do more thorough validation in other tests
			if !strings.Contains(rendered, "//") && !strings.Contains(rendered, "/*") && !strings.Contains(rendered, "#") {
				// Only fail if the original had comments
				if strings.Contains(tt.vcl, "//") || strings.Contains(tt.vcl, "/*") || strings.Contains(tt.vcl, "#") {
					t.Error("Rendered VCL does not contain any comments")
				}
			}
		})
	}
}

// TestCommentRoundTrip tests that parse -> render -> parse preserves comments
func TestCommentRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		vcl  string
	}{
		{
			name: "simple backend with comments",
			vcl: `vcl 4.0;

// Default backend configuration
backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`,
		},
		{
			name: "subroutine with comments",
			vcl: `vcl 4.0;

// Main request handler
sub vcl_recv {
    // Check for GET requests
    if (req.method == "GET") {
        return(hash);
    }
}`,
		},
		{
			name: "mixed comment styles",
			vcl: `vcl 4.0;

// Line comment
/* Block comment */
# Shell comment
backend default {
    .host = "127.0.0.1";
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
			rendered := renderer.Render(program1)
			t.Logf("Rendered VCL:\n%s", rendered)

			// Second parse
			program2, err := parser.Parse(rendered, "rendered.vcl")
			if err != nil {
				t.Fatalf("Second parse error on rendered VCL: %v\nRendered VCL:\n%s", err, rendered)
			}

			// Verify second parse succeeded
			if program2 == nil {
				t.Fatal("Second parse returned nil program")
			}

			// Render again
			rendered2 := renderer.Render(program2)
			t.Logf("Second render:\n%s", rendered2)

			// The two rendered versions should be identical
			if rendered != rendered2 {
				t.Errorf("Rendered versions differ after round-trip")
				t.Logf("First render:\n%s", rendered)
				t.Logf("Second render:\n%s", rendered2)
			}
		})
	}
}

// TestCommentAttachment tests that comments are attached to the correct nodes
func TestCommentAttachment(t *testing.T) {
	vcl := `vcl 4.0;

// Backend comment
backend default {
    .host = "127.0.0.1";
}

// Subroutine comment
sub vcl_recv {
    // Statement comment
    set req.http.Host = "example.com";
}`

	program, err := parser.Parse(vcl, "test.vcl")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Check that we have declarations
	if len(program.Declarations) != 2 {
		t.Fatalf("Expected 2 declarations, got %d", len(program.Declarations))
	}

	// Check backend has comment
	backend := program.Declarations[0]
	comments := backend.GetComments()
	if comments == nil || len(comments.Leading) == 0 {
		t.Error("Backend declaration should have leading comments")
	} else {
		if !strings.Contains(comments.Leading[0].Text, "Backend comment") {
			t.Errorf("Expected backend comment, got: %s", comments.Leading[0].Text)
		}
	}

	// Check subroutine has comment
	subDecl := program.Declarations[1]
	comments = subDecl.GetComments()
	if comments == nil || len(comments.Leading) == 0 {
		t.Error("Subroutine declaration should have leading comments")
	} else {
		if !strings.Contains(comments.Leading[0].Text, "Subroutine comment") {
			t.Errorf("Expected subroutine comment, got: %s", comments.Leading[0].Text)
		}
	}
}

// TestTrailingComments tests trailing comments on the same line
func TestTrailingComments(t *testing.T) {
	vcl := `vcl 4.0;

backend default { // main backend
    .host = "127.0.0.1";
}

sub vcl_recv {
    return(hash); // cache it
}`

	program, err := parser.Parse(vcl, "test.vcl")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Render and check for trailing comments
	rendered := renderer.Render(program)
	t.Logf("Rendered:\n%s", rendered)

	// Verify trailing comments are present
	if !strings.Contains(rendered, "// main backend") {
		t.Error("Missing trailing comment on backend declaration")
	}
	if !strings.Contains(rendered, "// cache it") {
		t.Error("Missing trailing comment on return statement")
	}
}

// TestBlockComments tests multi-line block comments
func TestBlockComments(t *testing.T) {
	vcl := `vcl 4.0;

/*
 * Multi-line block comment
 * describing the backend
 */
backend default {
    .host = "127.0.0.1";
}`

	program, err := parser.Parse(vcl, "test.vcl")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Check backend has block comment
	if len(program.Declarations) == 0 {
		t.Fatal("No declarations found")
	}

	backend := program.Declarations[0]
	comments := backend.GetComments()
	if comments == nil || len(comments.Leading) == 0 {
		t.Fatal("Backend should have leading comments")
	}

	// Check it's a block comment
	if !comments.Leading[0].IsBlock {
		t.Error("Expected block comment")
	}

	// Verify content
	if !strings.Contains(comments.Leading[0].Text, "Multi-line block comment") {
		t.Errorf("Block comment content incorrect: %s", comments.Leading[0].Text)
	}

	// Render and verify block comment is preserved
	rendered := renderer.Render(program)
	if !strings.Contains(rendered, "/*") || !strings.Contains(rendered, "*/") {
		t.Error("Block comment markers not found in rendered output")
	}
}

// TestComplexCommentScenario tests a realistic VCL file with various comments
func TestComplexCommentScenario(t *testing.T) {
	vcl := `// VCL Configuration for example.com
vcl 4.1;

// Import standard library
import std;

/*
 * Backend Definitions
 */

// Primary backend server
backend primary {
    .host = "192.168.1.10"; // production server
    .port = "8080";
}

// Fallback backend server
backend fallback {
    .host = "192.168.1.11";
    .port = "8080";
}

/*
 * Request Handling
 */

// Main request handler
sub vcl_recv {
    // Normalize host header
    set req.http.Host = regsub(req.http.Host, ":[0-9]+", "");

    // Handle POST requests
    if (req.method == "POST") {
        return(pass); // don't cache POSTs
    }

    // Cache GET and HEAD
    if (req.method == "GET" || req.method == "HEAD") {
        return(hash); // lookup in cache
    }

    // Pass everything else
    return(pass);
}`

	// Parse
	program, err := parser.Parse(vcl, "test.vcl")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Render
	rendered := renderer.Render(program)
	t.Logf("Rendered VCL:\n%s", rendered)

	// Verify key comments are present
	expectedComments := []string{
		"VCL Configuration",
		"Import standard library",
		"Backend Definitions",
		"Primary backend server",
		"production server",
		"Fallback backend server",
		"Request Handling",
		"Main request handler",
		"Normalize host header",
		"Handle POST requests",
		"don't cache POSTs",
		"Cache GET and HEAD",
		"lookup in cache",
		"Pass everything else",
	}

	for _, expected := range expectedComments {
		if !strings.Contains(rendered, expected) {
			t.Errorf("Missing expected comment: %s", expected)
		}
	}

	// Parse the rendered version
	program2, err := parser.Parse(rendered, "rendered.vcl")
	if err != nil {
		t.Fatalf("Failed to parse rendered VCL: %v", err)
	}

	// Render again and verify stability
	rendered2 := renderer.Render(program2)
	if rendered != rendered2 {
		t.Error("Rendered output not stable after round-trip")
	}
}
