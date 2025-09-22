package parser

import (
	"testing"

	"github.com/perbu/vclparser/pkg/lexer"
)

func TestMaxErrorsDefault(t *testing.T) {
	// Test that parser stops after 8 errors by default
	// Create VCL that will generate multiple errors by using expectPeek failures
	vclWithManyErrors := `vcl 4.0;
backend
backend
backend
backend
backend
backend
backend
backend
backend  // 9th error - should not be processed
backend`

	l := NewLexer(vclWithManyErrors, "test.vcl")
	p := New(l, vclWithManyErrors, "test.vcl")
	p.ParseProgram()

	// Should have exactly 8 errors (the default limit)
	if len(p.errors) > 8 {
		t.Errorf("Expected at most 8 errors (default limit), got %d", len(p.errors))
	}

	// Should be greater than 5 to show multiple errors were caught
	if len(p.errors) < 5 {
		t.Errorf("Expected multiple errors to demonstrate MaxErrors functionality, got %d", len(p.errors))
	}
}

func TestMaxErrorsUnlimited(t *testing.T) {
	// Test that MaxErrors=0 means unlimited
	vclWithManyErrors := `vcl 4.0;
backend
backend
backend
backend
backend
backend
backend
backend
backend
backend`

	config := &Config{
		MaxErrors: 0, // Unlimited
	}

	l := NewLexer(vclWithManyErrors, "test.vcl")
	p := NewWithConfig(l, vclWithManyErrors, "test.vcl", config)
	p.ParseProgram()

	// Should collect same or more errors than default case since no limit
	if len(p.errors) < 8 {
		t.Errorf("Expected multiple errors with unlimited setting, got %d", len(p.errors))
	}
}

func TestMaxErrorsCustomLimit(t *testing.T) {
	// Test custom limit of 3
	vclWithManyErrors := `vcl 4.0;
backend
backend
backend
backend
backend`

	config := &Config{
		MaxErrors: 3,
	}

	l := NewLexer(vclWithManyErrors, "test.vcl")
	p := NewWithConfig(l, vclWithManyErrors, "test.vcl", config)
	p.ParseProgram()

	if len(p.errors) > 3 {
		t.Errorf("Expected at most 3 errors (custom limit), got %d", len(p.errors))
	}

	if len(p.errors) < 2 {
		t.Errorf("Expected multiple errors to demonstrate functionality, got %d", len(p.errors))
	}
}

func TestMaxErrorsEarlyBailout(t *testing.T) {
	// Test basic MaxErrors functionality with default config
	vclWithErrors := `vcl 4.0;
backend  // Missing name - triggers error
backend  // Missing name - triggers error
backend default {
    .host = "localhost";
    .port = "8080";
}`

	// Test with very low MaxErrors limit
	config := &Config{MaxErrors: 1}
	l := NewLexer(vclWithErrors, "test.vcl")
	p := NewWithConfig(l, vclWithErrors, "test.vcl", config)
	program := p.ParseProgram()

	// Should stop after 1 error
	if len(p.errors) > 1 {
		t.Errorf("Expected at most 1 error with MaxErrors=1, got %d", len(p.errors))
	}

	// Should have minimal processing due to early bailout
	if len(program.Declarations) > 1 {
		t.Errorf("Expected minimal declarations due to early bailout, got %d", len(program.Declarations))
	}
}

// Helper function for lexer creation
func NewLexer(input, filename string) *lexer.Lexer {
	return lexer.New(input, filename)
}
