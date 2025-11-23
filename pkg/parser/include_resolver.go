package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
)

// includeResolver handles resolving include statements within the parser
type includeResolver struct {
	basePath     string
	maxDepth     int
	visitedFiles map[string]bool
	includeChain []string
	currentDepth int
}

// resolveIncludesInProgram resolves all include statements in a parsed program
func (p *Parser) resolveIncludesInProgram(program *ast.Program) (*ast.Program, error) {
	resolver := &includeResolver{
		basePath:     p.includeBasePath,
		maxDepth:     p.includeMaxDepth,
		visitedFiles: make(map[string]bool),
		includeChain: make([]string, 0),
		currentDepth: 0,
	}

	return resolver.processIncludes(program, p.filename)
}

// processIncludes walks through the AST and resolves include statements
func (r *includeResolver) processIncludes(program *ast.Program, currentFile string) (*ast.Program, error) {
	var newDeclarations []ast.Declaration

	for _, decl := range program.Declarations {
		if includeDecl, ok := decl.(*ast.IncludeDecl); ok {
			// Parse and resolve the included file
			includedProgram, err := r.resolveFile(includeDecl.Path, currentFile)
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

// resolveFile parses a single file and resolves its includes
func (r *includeResolver) resolveFile(filename, parentFile string) (*ast.Program, error) {
	// Check depth limit
	if r.currentDepth >= r.maxDepth {
		return nil, fmt.Errorf("maximum include depth exceeded (%d) while processing %s (chain: %v)",
			r.maxDepth, filename, append(r.includeChain, filename))
	}

	// Resolve path relative to parent file's directory
	var resolvedPath string
	if filepath.IsAbs(filename) {
		resolvedPath = filename
	} else {
		// If basePath is set, use it; otherwise use parent file's directory
		if r.basePath != "" {
			resolvedPath = filepath.Join(r.basePath, filename)
		} else {
			parentDir := filepath.Dir(parentFile)
			resolvedPath = filepath.Join(parentDir, filename)
		}
	}

	// Convert to absolute path for tracking
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path for %s: %w", filename, err)
	}

	// Check for circular includes
	if r.visitedFiles[absPath] {
		return nil, fmt.Errorf("circular include detected: %s (chain: %v)",
			filename, append(r.includeChain, filename))
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read include file %s: %w", filename, err)
	}

	// Mark this file as visited and add to chain
	r.visitedFiles[absPath] = true
	r.includeChain = append(r.includeChain, filename)
	r.currentDepth++

	// Parse the included file with appropriate options
	l := lexer.New(string(content), filename)
	p := New(l, string(content), filename,
		WithAllowMissingVersion(true),
		WithSkipSubroutineValidation(true),
	)
	program := p.ParseProgram()

	if len(p.errors) > 0 {
		return nil, fmt.Errorf("parse error in included file %s: %w", filename, p.errors[0])
	}

	// Recursively process includes in this file
	resolvedProgram, err := r.processIncludes(program, absPath)
	if err != nil {
		return nil, err
	}

	// Clean up state for this file
	r.currentDepth--
	r.includeChain = r.includeChain[:len(r.includeChain)-1]

	return resolvedProgram, nil
}
