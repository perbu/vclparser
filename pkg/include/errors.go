package include

import (
	"fmt"
	"strings"
)

// CircularIncludeError represents a circular include dependency
type CircularIncludeError struct {
	Path  string
	Chain []string
}

func (e *CircularIncludeError) Error() string {
	chain := strings.Join(e.Chain, " -> ")
	return fmt.Sprintf("circular include detected: %s (chain: %s)", e.Path, chain)
}

// MaxDepthError represents an include depth limit exceeded error
type MaxDepthError struct {
	Path     string
	MaxDepth int
	Current  int
}

func (e *MaxDepthError) Error() string {
	return fmt.Sprintf("maximum include depth exceeded for %s: %d (limit: %d)", e.Path, e.Current, e.MaxDepth)
}

// FileNotFoundError represents a missing include file error
type FileNotFoundError struct {
	Path     string
	BasePath string
	Cause    error
}

func (e *FileNotFoundError) Error() string {
	if e.BasePath != "" {
		return fmt.Sprintf("failed to read include file %s (base: %s): %v", e.Path, e.BasePath, e.Cause)
	}
	return fmt.Sprintf("failed to read include file %s: %v", e.Path, e.Cause)
}

func (e *FileNotFoundError) Unwrap() error {
	return e.Cause
}

// ParseError represents an error parsing an included file
type ParseError struct {
	Path  string
	Cause error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("failed to parse include file %s: %v", e.Path, e.Cause)
}

func (e *ParseError) Unwrap() error {
	return e.Cause
}
