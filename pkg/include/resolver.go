// Package include provides functionality for resolving VCL include statements.
//
// This package implements a two-phase approach to include resolution:
// 1. The parser creates IncludeDecl AST nodes without resolving them
// 2. This package walks the AST and resolves includes by merging included files
//
// This approach keeps the parser pure (no I/O) while providing flexible
// include resolution with proper error handling and circular dependency detection.
package include

import (
	"path/filepath"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

// Resolver handles parsing VCL files with include statements
type Resolver struct {
	fileReader   FileReader
	basePath     string
	visitedFiles map[string]bool
	includeChain []string
	maxDepth     int
	currentDepth int
}

// Option represents a configuration option for the Resolver
type Option func(*Resolver)

// WithBasePath sets the base path for resolving relative includes
func WithBasePath(basePath string) Option {
	return func(r *Resolver) {
		r.basePath = basePath
	}
}

// WithMaxDepth sets the maximum include depth (default: 10)
func WithMaxDepth(maxDepth int) Option {
	return func(r *Resolver) {
		r.maxDepth = maxDepth
	}
}

// WithFileReader sets a custom file reader (useful for testing)
func WithFileReader(reader FileReader) Option {
	return func(r *Resolver) {
		r.fileReader = reader
	}
}

// NewResolver creates a new include resolver with the given options
func NewResolver(options ...Option) *Resolver {
	resolver := &Resolver{
		visitedFiles: make(map[string]bool),
		includeChain: make([]string, 0),
		maxDepth:     10,
		currentDepth: 0,
	}

	// Apply options
	for _, option := range options {
		option(resolver)
	}

	// Set default file reader if none provided
	if resolver.fileReader == nil {
		resolver.fileReader = NewOSFileReader(resolver.basePath)
	}

	return resolver
}

// ResolveFile parses a VCL file and recursively resolves all include statements
func (r *Resolver) ResolveFile(filename string) (*ast.Program, error) {
	// Reset state for new resolution
	r.visitedFiles = make(map[string]bool)
	r.includeChain = make([]string, 0)
	r.currentDepth = 0

	return r.resolveFile(filename)
}

// Resolve takes an already-parsed program and resolves any include statements
func (r *Resolver) Resolve(program *ast.Program) (*ast.Program, error) {
	// Reset state
	r.visitedFiles = make(map[string]bool)
	r.includeChain = make([]string, 0)
	r.currentDepth = 0

	return r.processIncludes(program)
}

// resolveFile parses a single file and resolves its includes
func (r *Resolver) resolveFile(filename string) (*ast.Program, error) {
	// Check depth limit
	if r.currentDepth > r.maxDepth {
		return nil, &MaxDepthError{
			Path:     filename,
			MaxDepth: r.maxDepth,
			Current:  r.currentDepth,
		}
	}

	// Convert to absolute path for tracking
	absPath, err := filepath.Abs(filepath.Join(r.basePath, filename))
	if err != nil {
		return nil, &FileNotFoundError{
			Path:     filename,
			BasePath: r.basePath,
			Cause:    err,
		}
	}

	// Check for circular includes
	if r.visitedFiles[absPath] {
		return nil, &CircularIncludeError{
			Path:  filename,
			Chain: append(r.includeChain, filename),
		}
	}

	// Read the file
	content, err := r.fileReader.ReadFile(filename)
	if err != nil {
		return nil, &FileNotFoundError{
			Path:     filename,
			BasePath: r.basePath,
			Cause:    err,
		}
	}

	// Parse the file
	program, err := parser.Parse(string(content), filename)
	if err != nil {
		return nil, &ParseError{
			Path:  filename,
			Cause: err,
		}
	}

	// Mark this file as visited and add to chain
	r.visitedFiles[absPath] = true
	r.includeChain = append(r.includeChain, filename)
	r.currentDepth++

	// Process includes in this file
	resolvedProgram, err := r.processIncludes(program)
	if err != nil {
		return nil, err
	}

	// Clean up state for this file
	r.currentDepth--
	r.includeChain = r.includeChain[:len(r.includeChain)-1]

	return resolvedProgram, nil
}

// processIncludes walks through the AST and resolves include statements
func (r *Resolver) processIncludes(program *ast.Program) (*ast.Program, error) {
	var newDeclarations []ast.Declaration

	for _, decl := range program.Declarations {
		if includeDecl, ok := decl.(*ast.IncludeDecl); ok {
			// Parse the included file
			includedProgram, err := r.resolveFile(includeDecl.Path)
			if err != nil {
				return nil, err
			}

			// Add declarations from included file (preserving order)
			newDeclarations = append(newDeclarations, includedProgram.Declarations...)
		} else {
			// Keep non-include declarations
			newDeclarations = append(newDeclarations, decl)
		}
	}

	// Create new program with merged declarations
	mergedProgram := &ast.Program{
		BaseNode:     program.BaseNode,
		VCLVersion:   program.VCLVersion,
		Declarations: newDeclarations,
	}

	return mergedProgram, nil
}
