package vclparser_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

// IncludeResolver handles parsing VCL files with include statements
type IncludeResolver struct {
	basePath      string
	includedFiles map[string]bool
	maxDepth      int
	currentDepth  int
}

// NewIncludeResolver creates a new include resolver
func NewIncludeResolver(basePath string) *IncludeResolver {
	return &IncludeResolver{
		basePath:      basePath,
		includedFiles: make(map[string]bool),
		maxDepth:      10, // Prevent infinite recursion
	}
}

// ParseWithIncludes parses a VCL file and recursively resolves include statements
func (ir *IncludeResolver) ParseWithIncludes(filename string) (*ast.Program, error) {
	if ir.currentDepth > ir.maxDepth {
		return nil, fmt.Errorf("maximum include depth exceeded")
	}

	// Convert to absolute path for tracking
	absPath, err := filepath.Abs(filepath.Join(ir.basePath, filename))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve path for %s: %v", filename, err)
	}

	// Check for circular includes
	if ir.includedFiles[absPath] {
		return nil, fmt.Errorf("circular include detected: %s", filename)
	}

	// Read the file
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %v", filename, err)
	}

	// Parse the main file
	program, err := parser.Parse(string(content), filename)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s: %v", filename, err)
	}

	// Mark this file as included
	ir.includedFiles[absPath] = true
	ir.currentDepth++

	// Process include statements
	mergedProgram, err := ir.processIncludes(program)
	if err != nil {
		return nil, err
	}

	ir.currentDepth--
	return mergedProgram, nil
}

// processIncludes walks through the AST and resolves include statements
func (ir *IncludeResolver) processIncludes(program *ast.Program) (*ast.Program, error) {
	var newDeclarations []ast.Declaration

	for _, decl := range program.Declarations {
		if includeDecl, ok := decl.(*ast.IncludeDecl); ok {
			// Parse the included file
			includedProgram, err := ir.ParseWithIncludes(includeDecl.Path)
			if err != nil {
				return nil, fmt.Errorf("failed to process include %s: %v", includeDecl.Path, err)
			}

			// Add declarations from included file
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

// countDeclarationsByType counts declarations by type for verification
func countDeclarationsByType(program *ast.Program) map[string]int {
	counts := make(map[string]int)

	for _, decl := range program.Declarations {
		switch decl.(type) {
		case *ast.BackendDecl:
			counts["backend"]++
		case *ast.SubDecl:
			counts["subroutine"]++
		case *ast.ACLDecl:
			counts["acl"]++
		case *ast.ImportDecl:
			counts["import"]++
		case *ast.IncludeDecl:
			counts["include"]++
		default:
			counts["other"]++
		}
	}

	return counts
}

// findDeclarationByName finds a declaration by name for verification
func findDeclarationByName(program *ast.Program, declType, name string) ast.Declaration {
	for _, decl := range program.Declarations {
		switch d := decl.(type) {
		case *ast.BackendDecl:
			if declType == "backend" && d.Name == name {
				return d
			}
		case *ast.SubDecl:
			if declType == "subroutine" && d.Name == name {
				return d
			}
		case *ast.ACLDecl:
			if declType == "acl" && d.Name == name {
				return d
			}
		}
	}
	return nil
}

func TestIncludeIntegration(t *testing.T) {
	testDataDir := filepath.Join("testdata", "includes")
	t.Run("CoreParserIncludeLimitation", func(t *testing.T) {
		mainVCLPath := filepath.Join(testDataDir, "main.vcl")
		content, err := os.ReadFile(mainVCLPath)
		if err != nil {
			t.Fatalf("Failed to read main.vcl: %v", err)
		}

		program, err := parser.Parse(string(content), "main.vcl")
		if err != nil {
			t.Fatalf("Core parser failed to parse main.vcl: %v", err)
		}

		// Count include declarations - they should still be present
		counts := countDeclarationsByType(program)
		if counts["include"] == 0 {
			t.Error("Core parser resolved includes automatically - this test needs updating!")
		}
		// Verify we only have declarations from main.vcl, not included files
		expectedSubroutines := 2 // Only from main.vcl
		if counts["subroutine"] != expectedSubroutines {
			t.Logf("Core parser found %d subroutines (expected %d from main.vcl only)",
				counts["subroutine"], expectedSubroutines)
		}

		// Should not find declarations from included files
		web1Backend := findDeclarationByName(program, "backend", "web1")
		if web1Backend != nil {
			t.Error("UNEXPECTED: Found backend from included file - core parser may have resolved includes!")
		}
	})

	t.Run("CustomIncludeResolver", func(t *testing.T) {
		// Now demonstrate how to properly handle includes with custom resolver
		resolver := NewIncludeResolver(testDataDir)
		program, err := resolver.ParseWithIncludes("main.vcl")

		if err != nil {
			t.Fatalf("Custom include resolver failed: %v", err)
		}

		// This demonstrates the INTENDED behavior for include resolution
		// When the core parser supports includes, this custom resolver won't be needed

		// Verify the main VCL version is preserved
		if program.VCLVersion == nil || program.VCLVersion.Version != "4.1" {
			t.Errorf("Expected VCL version 4.1, got %v", program.VCLVersion)
		}

		// Count declarations to verify includes were processed
		counts := countDeclarationsByType(program)

		// Expected counts from our test files:
		// backends.vcl: 3 backends + 1 import + 1 subroutine (vcl_init)
		// subroutines.vcl: 3 subroutines
		// acls.vcl: 3 ACLs
		// main.vcl: 2 subroutines
		expectedBackends := 3
		expectedSubroutines := 6 // 3 from subroutines.vcl + 1 from backends.vcl + 2 from main.vcl
		expectedACLs := 3
		expectedImports := 1

		if counts["backend"] != expectedBackends {
			t.Errorf("Expected %d backends, got %d", expectedBackends, counts["backend"])
		}
		if counts["subroutine"] != expectedSubroutines {
			t.Errorf("Expected %d subroutines, got %d", expectedSubroutines, counts["subroutine"])
		}
		if counts["acl"] != expectedACLs {
			t.Errorf("Expected %d ACLs, got %d", expectedACLs, counts["acl"])
		}
		if counts["import"] != expectedImports {
			t.Errorf("Expected %d imports, got %d", expectedImports, counts["import"])
		}

		// Verify specific declarations exist
		web1Backend := findDeclarationByName(program, "backend", "web1")
		if web1Backend == nil {
			t.Error("Expected to find web1 backend from backends.vcl")
		}

		normalizeHeadersSub := findDeclarationByName(program, "subroutine", "normalize_headers")
		if normalizeHeadersSub == nil {
			t.Error("Expected to find normalize_headers subroutine from subroutines.vcl")
		}

		internalIpsACL := findDeclarationByName(program, "acl", "internal_ips")
		if internalIpsACL == nil {
			t.Error("Expected to find internal_ips ACL from acls.vcl")
		}

		// Verify no include declarations remain (they should be replaced)
		if counts["include"] != 0 {
			t.Errorf("Expected 0 include declarations after processing, got %d", counts["include"])
		}
	})

	t.Run("NestedIncludes", func(t *testing.T) {
		resolver := NewIncludeResolver(testDataDir)
		program, err := resolver.ParseWithIncludes("nested_main.vcl")

		if err != nil {
			t.Fatalf("Failed to parse nested includes: %v", err)
		}

		counts := countDeclarationsByType(program)

		// Expected: 2 backends + 1 ACL + 3 subroutines
		if counts["backend"] != 2 {
			t.Errorf("Expected 2 backends from nested includes, got %d", counts["backend"])
		}
		if counts["acl"] != 1 {
			t.Errorf("Expected 1 ACL from nested includes, got %d", counts["acl"])
		}
		if counts["subroutine"] != 3 {
			t.Errorf("Expected 3 subroutines from nested includes, got %d", counts["subroutine"])
		}

		// Verify deep nesting worked
		level2Backend := findDeclarationByName(program, "backend", "level2_backend")
		if level2Backend == nil {
			t.Error("Expected to find level2_backend from deeply nested include")
		}
	})

	t.Run("CircularIncludeDetection", func(t *testing.T) {
		resolver := NewIncludeResolver(testDataDir)
		_, err := resolver.ParseWithIncludes("circular1.vcl")
		if err == nil {
			t.Fatal("Expected circular include detection to fail, but it succeeded")
		}
		if !strings.Contains(err.Error(), "circular include") {
			t.Errorf("Expected circular include error, got: %v", err)
		}
	})

	t.Run("MissingFileHandling", func(t *testing.T) {
		// Create a temporary VCL file that includes a non-existent file
		tmpDir := t.TempDir()
		invalidVCL := `vcl 4.0;
include "nonexistent.vcl";
`
		tmpFile := filepath.Join(tmpDir, "invalid.vcl")
		err := os.WriteFile(tmpFile, []byte(invalidVCL), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		resolver := NewIncludeResolver(tmpDir)
		_, err = resolver.ParseWithIncludes("invalid.vcl")

		if err == nil {
			t.Fatal("Expected missing file error, but parsing succeeded")
		}

		if !strings.Contains(err.Error(), "failed to read file") {
			t.Errorf("Expected missing file error, got: %v", err)
		}

	})
}

// TestIncludeResolverExample demonstrates how to use the include resolver
func TestIncludeResolverExample(t *testing.T) {
	testDataDir := filepath.Join("testdata", "includes")
	// Example: Parse VCL with includes and inspect the result
	resolver := NewIncludeResolver(testDataDir)
	if _, err := resolver.ParseWithIncludes("main.vcl"); err != nil {
		t.Fatalf("Example failed: %v", err)
	}
}

// BenchmarkIncludeResolution benchmarks the include resolution performance
func BenchmarkIncludeResolution(b *testing.B) {
	testDataDir := filepath.Join("testdata", "includes")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver := NewIncludeResolver(testDataDir)
		_, err := resolver.ParseWithIncludes("main.vcl")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
