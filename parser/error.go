package parser

import (
	"fmt"
	"strings"

	"github.com/varnish/vclparser/lexer"
)

// DetailedError represents a parsing error with enhanced context visualization
type DetailedError struct {
	Message  string
	Position lexer.Position
	Token    lexer.Token
	Filename string
	Source   string // Full VCL source code for context
}

// Error implements the error interface with rich context formatting
func (e DetailedError) Error() string {
	var result strings.Builder

	// Header with filename and position
	result.WriteString(fmt.Sprintf("Parse error in %s at line %d:%d\n",
		e.Filename, e.Position.Line, e.Position.Column))

	// Get context lines
	lines := strings.Split(e.Source, "\n")
	if len(lines) == 0 {
		result.WriteString(fmt.Sprintf("Error: %s", e.Message))
		return result.String()
	}

	errorLine := e.Position.Line - 1 // Convert to 0-indexed

	// Show line before error (if exists)
	if errorLine > 0 {
		result.WriteString(fmt.Sprintf("%3d | %s\n", errorLine, lines[errorLine-1]))
	}

	// Show error line
	if errorLine >= 0 && errorLine < len(lines) {
		result.WriteString(fmt.Sprintf("%3d | %s\n", errorLine+1, lines[errorLine]))

		// Add caret pointer to exact error position
		spaces := strings.Repeat(" ", 6+e.Position.Column-1) // "nnn | " + column offset
		result.WriteString(fmt.Sprintf("%s^\n", spaces))
	}

	// Show line after error (if exists)
	if errorLine+1 < len(lines) {
		result.WriteString(fmt.Sprintf("%3d | %s\n", errorLine+2, lines[errorLine+1]))
	}

	// Add blank line and error message
	result.WriteString(fmt.Sprintf("\nError: %s\n", e.Message))

	return result.String()
}

// ParseError represents a basic parsing error (for backward compatibility)
type ParseError struct {
	Message  string
	Position lexer.Position
	Token    lexer.Token
}

func (e ParseError) Error() string {
	return fmt.Sprintf("parse error at %s: %s (got %s)", e.Position, e.Message, e.Token.Type)
}
