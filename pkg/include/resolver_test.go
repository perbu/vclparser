package include

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

// Test helper functions

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

// Test memory file reader setup

func createTestFiles() *MemoryFileReader {
	files := map[string]string{
		"main.vcl": `vcl 4.1;

include "backends.vcl";
include "subroutines.vcl";
include "acls.vcl";

sub vcl_recv {
    if (client.ip ~ internal_ips) {
        set req.backend_hint = web_cluster;
    }
}

sub vcl_backend_response {
    set beresp.ttl = 1h;
}`,
		"backends.vcl": `vcl 4.1;

import std;

backend web1 {
    .host = "web1.example.com";
    .port = "80";
}

backend web2 {
    .host = "web2.example.com";
    .port = "80";
}

backend web_cluster {
    .host = "cluster.example.com";
    .port = "80";
}

sub vcl_init {
    return (ok);
}`,
		"subroutines.vcl": `vcl 4.1;

sub normalize_headers {
    unset req.http.X-Internal;
}

sub log_request {
    std.log("Request: " + req.url);
}

sub handle_error {
    set obj.status = 503;
}`,
		"acls.vcl": `vcl 4.1;

acl internal_ips {
    "192.168.1.0"/24;
    "10.0.0.0"/8;
}

acl admin_ips {
    "192.168.1.100";
}

acl public_ips {
    !"192.168.0.0"/16;
    !"10.0.0.0"/8;
}`,
		"nested_main.vcl": `vcl 4.0;
include "nested_level1.vcl";`,
		"nested_level1.vcl": `vcl 4.0;
include "nested_level2.vcl";
sub level1_sub {
    return (lookup);
}`,
		"nested_level2.vcl": `vcl 4.0;

backend level2_backend {
    .host = "level2.example.com";
}

acl level2_acl {
    "172.16.0.0"/12;
}

sub level2_sub {
    return (pass);
}`,
		"circular1.vcl": `vcl 4.0;
include "circular2.vcl";`,
		"circular2.vcl": `vcl 4.0;
include "circular1.vcl";`,
	}
	return NewMemoryFileReader(files)
}

// Unit tests

func TestResolver_Basic(t *testing.T) {
	reader := createTestFiles()
	resolver := NewResolver(WithFileReader(reader))

	program, err := resolver.ResolveFile("main.vcl")
	if err != nil {
		t.Fatalf("Failed to resolve includes: %v", err)
	}

	// Verify VCL version is preserved
	if program.VCLVersion == nil || program.VCLVersion.Version != "4.1" {
		t.Errorf("Expected VCL version 4.1, got %v", program.VCLVersion)
	}

	// Count declarations
	counts := countDeclarationsByType(program)

	// Expected: 3 backends + 1 import + 6 subroutines + 3 ACLs
	expectedBackends := 3
	expectedSubroutines := 6 // 4 from includes + 2 from main
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
	if findDeclarationByName(program, "backend", "web1") == nil {
		t.Error("Expected to find web1 backend")
	}
	if findDeclarationByName(program, "subroutine", "normalize_headers") == nil {
		t.Error("Expected to find normalize_headers subroutine")
	}
	if findDeclarationByName(program, "acl", "internal_ips") == nil {
		t.Error("Expected to find internal_ips ACL")
	}

	// Verify no include declarations remain
	if counts["include"] != 0 {
		t.Errorf("Expected 0 include declarations after processing, got %d", counts["include"])
	}
}

func TestResolver_NestedIncludes(t *testing.T) {
	reader := createTestFiles()
	resolver := NewResolver(WithFileReader(reader))

	program, err := resolver.ResolveFile("nested_main.vcl")
	if err != nil {
		t.Fatalf("Failed to resolve nested includes: %v", err)
	}

	counts := countDeclarationsByType(program)

	// Expected: 1 backend + 1 ACL + 2 subroutines
	if counts["backend"] != 1 {
		t.Errorf("Expected 1 backend from nested includes, got %d", counts["backend"])
	}
	if counts["acl"] != 1 {
		t.Errorf("Expected 1 ACL from nested includes, got %d", counts["acl"])
	}
	if counts["subroutine"] != 2 {
		t.Errorf("Expected 2 subroutines from nested includes, got %d", counts["subroutine"])
	}

	// Verify deep nesting worked
	if findDeclarationByName(program, "backend", "level2_backend") == nil {
		t.Error("Expected to find level2_backend from deeply nested include")
	}
}

func TestResolver_CircularIncludeDetection(t *testing.T) {
	reader := createTestFiles()
	resolver := NewResolver(WithFileReader(reader))

	_, err := resolver.ResolveFile("circular1.vcl")
	if err == nil {
		t.Fatal("Expected circular include detection to fail, but it succeeded")
	}

	if !strings.Contains(err.Error(), "circular include") {
		t.Errorf("Expected circular include error, got: %v", err)
	}

	// Check if it's the right error type
	if circularErr, ok := err.(*CircularIncludeError); ok {
		if len(circularErr.Chain) == 0 {
			t.Error("Expected include chain in circular error")
		}
		t.Logf("Detected circular include with chain: %v", circularErr.Chain)
	}
}

func TestResolver_MissingFile(t *testing.T) {
	reader := NewMemoryFileReader(map[string]string{
		"main.vcl": `vcl 4.0;
include "nonexistent.vcl";`,
	})
	resolver := NewResolver(WithFileReader(reader))

	_, err := resolver.ResolveFile("main.vcl")
	if err == nil {
		t.Fatal("Expected missing file error, but parsing succeeded")
	}

	if fileErr, ok := err.(*FileNotFoundError); !ok {
		t.Errorf("Expected FileNotFoundError, got %T: %v", err, err)
	} else {
		if fileErr.Path != "nonexistent.vcl" {
			t.Errorf("Expected path 'nonexistent.vcl', got '%s'", fileErr.Path)
		}
		t.Logf("Correctly detected missing file: %s", fileErr.Path)
	}
}

func TestResolver_MaxDepth(t *testing.T) {
	// Create a chain of includes that exceeds max depth
	files := make(map[string]string)
	files["main.vcl"] = `vcl 4.0;
include "level1.vcl";`

	for i := 1; i <= 15; i++ {
		files[fmt.Sprintf("level%d.vcl", i)] = fmt.Sprintf(`vcl 4.0;
include "level%d.vcl";`, i+1)
	}

	reader := NewMemoryFileReader(files)
	resolver := NewResolver(WithFileReader(reader), WithMaxDepth(5))

	_, err := resolver.ResolveFile("main.vcl")
	if err == nil {
		t.Fatal("Expected max depth error, but parsing succeeded")
	}

	if depthErr, ok := err.(*MaxDepthError); !ok {
		t.Errorf("Expected MaxDepthError, got %T: %v", err, err)
	} else {
		if depthErr.MaxDepth != 5 {
			t.Errorf("Expected max depth 5, got %d", depthErr.MaxDepth)
		}
		t.Logf("Correctly detected max depth exceeded: %d/%d", depthErr.Current, depthErr.MaxDepth)
	}
}

func TestResolver_ParseError(t *testing.T) {
	reader := NewMemoryFileReader(map[string]string{
		"main.vcl": `vcl 4.0;
include "invalid.vcl";`,
		"invalid.vcl": `invalid VCL syntax here!`,
	})
	resolver := NewResolver(WithFileReader(reader))

	_, err := resolver.ResolveFile("main.vcl")
	if err == nil {
		t.Fatal("Expected parse error, but parsing succeeded")
	}

	if parseErr, ok := err.(*ParseError); !ok {
		t.Errorf("Expected ParseError, got %T: %v", err, err)
	} else {
		if parseErr.Path != "invalid.vcl" {
			t.Errorf("Expected path 'invalid.vcl', got '%s'", parseErr.Path)
		}
		t.Logf("Correctly detected parse error in: %s", parseErr.Path)
	}
}

func TestResolver_ResolveProgram(t *testing.T) {
	// First parse a program with includes
	reader := createTestFiles()
	content, _ := reader.ReadFile("main.vcl")
	program, err := parser.Parse(string(content), "main.vcl")
	if err != nil {
		t.Fatalf("Failed to parse main.vcl: %v", err)
	}

	// Verify it has include declarations
	counts := countDeclarationsByType(program)
	if counts["include"] == 0 {
		t.Fatal("Expected include declarations in parsed program")
	}

	// Now resolve the includes
	resolver := NewResolver(WithFileReader(reader))
	resolved, err := resolver.Resolve(program)
	if err != nil {
		t.Fatalf("Failed to resolve program: %v", err)
	}

	// Verify includes were resolved
	resolvedCounts := countDeclarationsByType(resolved)
	if resolvedCounts["include"] != 0 {
		t.Errorf("Expected 0 include declarations after resolution, got %d", resolvedCounts["include"])
	}

	// Should have more declarations than before
	if len(resolved.Declarations) <= len(program.Declarations) {
		t.Error("Expected more declarations after include resolution")
	}
}

// API function tests

func TestAPI_ResolveFile(t *testing.T) {
	// This test uses real files, so we need to check if test data exists
	testDataDir := filepath.Join("..", "..", "tests", "testdata", "includes")
	mainFile := filepath.Join(testDataDir, "main.vcl")

	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Skip("Test data directory not found, skipping integration test")
	}

	program, err := ResolveFileWithBasePath("main.vcl", testDataDir)
	if err != nil {
		t.Fatalf("API ResolveFileWithBasePath failed: %v", err)
	}

	counts := countDeclarationsByType(program)
	if counts["include"] != 0 {
		t.Errorf("Expected 0 include declarations, got %d", counts["include"])
	}

	// Should have backends, subroutines, and ACLs from includes
	if counts["backend"] == 0 || counts["subroutine"] == 0 || counts["acl"] == 0 {
		t.Error("Expected declarations from included files")
	}
}

func TestAPI_ResolveProgram(t *testing.T) {
	reader := createTestFiles()
	content, _ := reader.ReadFile("main.vcl")
	program, err := parser.Parse(string(content), "main.vcl")
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	// Use API function (this will use OS file reader, so we can't test with memory)
	// Just test that the function exists and has the right signature
	_, err = ResolveProgram(program)
	// This will fail because we don't have real files, but that's OK
	// We just want to test the API exists
	if err == nil {
		t.Log("Unexpected success - this should fail with missing files")
	}
}

// Benchmark tests

func BenchmarkResolver_Simple(b *testing.B) {
	reader := createTestFiles()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver := NewResolver(WithFileReader(reader))
		_, err := resolver.ResolveFile("main.vcl")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkResolver_Nested(b *testing.B) {
	reader := createTestFiles()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resolver := NewResolver(WithFileReader(reader))
		_, err := resolver.ResolveFile("nested_main.vcl")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
