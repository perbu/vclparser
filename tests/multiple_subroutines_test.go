package vclparser_test

import (
	"testing"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

// TestMultipleSubroutineDefinitions tests that VCL allows multiple definitions
// of the same subroutine, which should be merged in order
func TestMultipleSubroutineDefinitions(t *testing.T) {
	input := `vcl 4.1;

sub vcl_recv {
	set req.http.X-First = "1";
}

sub vcl_recv {
	set req.http.X-Second = "2";
}
`

	program, err := parser.Parse(input, "test.vcl")
	if err != nil {
		t.Fatalf("Parser returned error: %v", err)
	}

	// Should have exactly one vcl_recv declaration
	var vclRecvCount int
	var vclRecvDecl *ast.SubDecl
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok && subDecl.Name == "vcl_recv" {
			vclRecvCount++
			vclRecvDecl = subDecl
		}
	}

	if vclRecvCount != 1 {
		t.Errorf("Expected 1 vcl_recv declaration after merging, got %d", vclRecvCount)
	}

	// Should have 2 statements in the merged body
	if vclRecvDecl == nil {
		t.Fatal("vcl_recv declaration is nil")
	}

	if vclRecvDecl.Body == nil {
		t.Fatal("vcl_recv body is nil")
	}

	stmtCount := len(vclRecvDecl.Body.Statements)
	if stmtCount != 2 {
		t.Errorf("Expected 2 statements in merged vcl_recv body, got %d", stmtCount)
	}
}

// TestMultipleSubroutinesOrderPreserved verifies that statements are appended
// in the order the subroutine definitions appear
func TestMultipleSubroutinesOrderPreserved(t *testing.T) {
	input := `vcl 4.1;

sub vcl_recv {
	set req.http.X-First = "1";
}

sub vcl_backend_fetch {
	set bereq.http.X-Backend = "backend";
}

sub vcl_recv {
	set req.http.X-Second = "2";
}

sub vcl_recv {
	set req.http.X-Third = "3";
}
`

	program, err := parser.Parse(input, "test.vcl")
	if err != nil {
		t.Fatalf("Parser returned error: %v", err)
	}

	// Find vcl_recv and vcl_backend_fetch
	var vclRecv *ast.SubDecl
	var vclBackendFetch *ast.SubDecl
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok {
			if subDecl.Name == "vcl_recv" {
				vclRecv = subDecl
			} else if subDecl.Name == "vcl_backend_fetch" {
				vclBackendFetch = subDecl
			}
		}
	}

	// Verify vcl_recv has 3 statements in order
	if vclRecv == nil {
		t.Fatal("vcl_recv not found")
	}
	if len(vclRecv.Body.Statements) != 3 {
		t.Errorf("Expected 3 statements in vcl_recv, got %d", len(vclRecv.Body.Statements))
	}

	// Verify vcl_backend_fetch has 1 statement
	if vclBackendFetch == nil {
		t.Fatal("vcl_backend_fetch not found")
	}
	if len(vclBackendFetch.Body.Statements) != 1 {
		t.Errorf("Expected 1 statement in vcl_backend_fetch, got %d", len(vclBackendFetch.Body.Statements))
	}

	// Verify all statements are SetStatements (order is preserved by having 3 statements)
	for i, stmt := range vclRecv.Body.Statements {
		if _, ok := stmt.(*ast.SetStatement); !ok {
			t.Errorf("Statement %d is not a SetStatement, got %T", i, stmt)
		}
	}
}

// TestEmptySubroutineMerge tests merging when one subroutine definition is empty
func TestEmptySubroutineMerge(t *testing.T) {
	input := `vcl 4.1;

sub vcl_recv {
	set req.http.X-First = "1";
}

sub vcl_recv {
	// Empty body
}

sub vcl_recv {
	set req.http.X-Second = "2";
}
`

	program, err := parser.Parse(input, "test.vcl")
	if err != nil {
		t.Fatalf("Parser returned error: %v", err)
	}

	var vclRecv *ast.SubDecl
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok && subDecl.Name == "vcl_recv" {
			vclRecv = subDecl
		}
	}

	if vclRecv == nil {
		t.Fatal("vcl_recv not found")
	}

	// Should have 2 statements (empty body doesn't add anything)
	stmtCount := len(vclRecv.Body.Statements)
	if stmtCount != 2 {
		t.Errorf("Expected 2 statements (empty body ignored), got %d", stmtCount)
	}
}

// TestUserDefinedSubroutineMerge tests that user-defined subroutines can also be merged
func TestUserDefinedSubroutineMerge(t *testing.T) {
	input := `vcl 4.1;

sub my_custom_sub {
	set req.http.X-Custom1 = "1";
}

sub my_custom_sub {
	set req.http.X-Custom2 = "2";
}
`

	program, err := parser.Parse(input, "test.vcl")
	if err != nil {
		t.Fatalf("Parser returned error: %v", err)
	}

	var customSub *ast.SubDecl
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok && subDecl.Name == "my_custom_sub" {
			customSub = subDecl
		}
	}

	if customSub == nil {
		t.Fatal("my_custom_sub not found")
	}

	stmtCount := len(customSub.Body.Statements)
	if stmtCount != 2 {
		t.Errorf("Expected 2 statements in merged user-defined subroutine, got %d", stmtCount)
	}
}

// TestManyMergedSubroutines tests merging many (more than 2) definitions
func TestManyMergedSubroutines(t *testing.T) {
	input := `vcl 4.1;

sub vcl_recv {
	set req.http.X-1 = "1";
}

sub vcl_recv {
	set req.http.X-2 = "2";
}

sub vcl_recv {
	set req.http.X-3 = "3";
}

sub vcl_recv {
	set req.http.X-4 = "4";
}

sub vcl_recv {
	set req.http.X-5 = "5";
}
`

	program, err := parser.Parse(input, "test.vcl")
	if err != nil {
		t.Fatalf("Parser returned error: %v", err)
	}

	var vclRecv *ast.SubDecl
	declarationCount := 0
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok && subDecl.Name == "vcl_recv" {
			vclRecv = subDecl
			declarationCount++
		}
	}

	if declarationCount != 1 {
		t.Errorf("Expected 1 vcl_recv declaration in program.Declarations, got %d", declarationCount)
	}

	if vclRecv == nil {
		t.Fatal("vcl_recv not found")
	}

	stmtCount := len(vclRecv.Body.Statements)
	if stmtCount != 5 {
		t.Errorf("Expected 5 statements after merging 5 definitions, got %d", stmtCount)
	}
}
