package parser

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/lexer"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.DisableInlineC {
		t.Error("Expected DisableInlineC to be false by default")
	}
	if config.MaxErrors != 8 {
		t.Errorf("Expected MaxErrors to be 8 by default, got %d", config.MaxErrors)
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
	config := &Config{
		DisableInlineC: true,
		MaxErrors:      0,
	}
	_, err = ParseWithConfig(vclWithInlineC, "test.vcl", config)
	if err == nil {
		t.Error("Expected parse to fail with inline C disabled")
	}
	if !strings.Contains(err.Error(), "inline C code blocks are disabled") {
		t.Errorf("Expected error message about inline C being disabled, got: %v", err)
	}
}

func TestNewWithConfig(t *testing.T) {
	config := &Config{
		DisableInlineC: true,
		MaxErrors:      10,
	}

	l := lexer.New("vcl 4.1;", "test.vcl")
	p := NewWithConfig(l, "vcl 4.1;", "test.vcl", config)

	if !p.config.DisableInlineC {
		t.Error("Expected parser to have DisableInlineC enabled")
	}
	if p.config.MaxErrors != 10 {
		t.Errorf("Expected parser to have MaxErrors=10, got %d", p.config.MaxErrors)
	}
}

func TestNewWithConfigNil(t *testing.T) {
	l := lexer.New("vcl 4.1;", "test.vcl")
	p := NewWithConfig(l, "vcl 4.1;", "test.vcl", nil)

	// Should use default config when nil is passed
	if p.config.DisableInlineC {
		t.Error("Expected parser to use default config when nil passed")
	}
	if p.config.MaxErrors != 8 {
		t.Errorf("Expected parser to use default MaxErrors=8, got %d", p.config.MaxErrors)
	}
}
