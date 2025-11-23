package parser

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/lexer"
)

func TestDefaultOptions(t *testing.T) {
	l := lexer.New("vcl 4.1;", "test.vcl")
	p := New(l, "vcl 4.1;", "test.vcl")

	if p.disableInlineC {
		t.Error("Expected disableInlineC to be false by default")
	}
	if p.maxErrors != 8 {
		t.Errorf("Expected maxErrors to be 8 by default, got %d", p.maxErrors)
	}
	if p.includeMaxDepth != 10 {
		t.Errorf("Expected includeMaxDepth to be 10 by default, got %d", p.includeMaxDepth)
	}
}

func TestDisableInlineC(t *testing.T) {
	vclWithInlineC := `vcl 4.1;
sub vcl_recv {
    C{
        printf("Hello from C!\n");
    }C
}`

	// Test with inline C enabled (default)
	_, err := Parse(vclWithInlineC, "test.vcl")
	if err != nil {
		t.Errorf("Expected parse to succeed with inline C enabled, got: %v", err)
	}

	// Test with inline C disabled
	_, err = Parse(vclWithInlineC, "test.vcl", WithDisableInlineC(true))
	if err == nil {
		t.Error("Expected parse to fail with inline C disabled")
	}
	if !strings.Contains(err.Error(), "inline C code blocks are disabled") {
		t.Errorf("Expected error message about inline C being disabled, got: %v", err)
	}
}

func TestWithMaxErrors(t *testing.T) {
	l := lexer.New("vcl 4.1;", "test.vcl")
	p := New(l, "vcl 4.1;", "test.vcl", WithMaxErrors(10))

	if p.maxErrors != 10 {
		t.Errorf("Expected parser to have maxErrors=10, got %d", p.maxErrors)
	}
}

func TestWithAllowMissingVersion(t *testing.T) {
	vclWithoutVersion := `backend default {
    .host = "127.0.0.1";
    .port = "8080";
}`

	// Test without option (should fail)
	_, err := Parse(vclWithoutVersion, "test.vcl")
	if err == nil {
		t.Error("Expected parse to fail without version declaration")
	}

	// Test with option (should succeed)
	_, err = Parse(vclWithoutVersion, "test.vcl", WithAllowMissingVersion(true))
	if err != nil {
		t.Errorf("Expected parse to succeed with WithAllowMissingVersion, got: %v", err)
	}
}

func TestMultipleOptions(t *testing.T) {
	l := lexer.New("vcl 4.1;", "test.vcl")
	p := New(l, "vcl 4.1;", "test.vcl",
		WithMaxErrors(5),
		WithDisableInlineC(true),
		WithAllowMissingVersion(true),
		WithSkipSubroutineValidation(true),
	)

	if p.maxErrors != 5 {
		t.Errorf("Expected maxErrors=5, got %d", p.maxErrors)
	}
	if !p.disableInlineC {
		t.Error("Expected disableInlineC to be true")
	}
	if !p.allowMissingVersion {
		t.Error("Expected allowMissingVersion to be true")
	}
	if !p.skipSubroutineValidation {
		t.Error("Expected skipSubroutineValidation to be true")
	}
}

func TestWithResolveIncludes(t *testing.T) {
	l := lexer.New("vcl 4.1;", "test.vcl")
	p := New(l, "vcl 4.1;", "test.vcl",
		WithResolveIncludes("/etc/varnish"),
		WithIncludeMaxDepth(5),
	)

	if !p.resolveIncludes {
		t.Error("Expected resolveIncludes to be true")
	}
	if p.includeBasePath != "/etc/varnish" {
		t.Errorf("Expected includeBasePath='/etc/varnish', got %s", p.includeBasePath)
	}
	if p.includeMaxDepth != 5 {
		t.Errorf("Expected includeMaxDepth=5, got %d", p.includeMaxDepth)
	}
}
