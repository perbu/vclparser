package analyzer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
	"github.com/varnish/vclparser/parser"
	"github.com/varnish/vclparser/types"
	"github.com/varnish/vclparser/vcc"
	"github.com/varnish/vclparser/vmod"
)

func setupTestRegistry(t *testing.T) *vmod.Registry {
	registry := vmod.NewRegistry()

	// Create temporary directory for VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	// Create std.vcc
	stdVCC := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING_LIST s)
$Function VOID log(STRING_LIST s)
$Function REAL random(REAL lo, REAL hi)
$Function BOOL file_exists(STRING path)`

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(stdVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	// Create directors.vcc
	directorsVCC := `$Module directors 3 "Directors module"
$ABI strict

$Object round_robin()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()

$Object hash()
$Method VOID .add_backend(BACKEND backend, REAL weight)
$Method BACKEND .backend(STRING key)`

	directorsFile := filepath.Join(tmpDir, "directors.vcc")
	err = os.WriteFile(directorsFile, []byte(directorsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write directors.vcc: %v", err)
	}

	// Load the VCC files
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC files: %v", err)
	}

	return registry
}

func parseVCL(t *testing.T, vclCode string) *ast.Program {
	// Use lexer and parser directly to avoid import cycle
	l := lexer.New(vclCode, "test.vcl")
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
	}
	return program
}

func TestValidateImport(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test valid import
	vclCode := `vcl 4.0;
import std;`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	if len(errors) != 0 {
		t.Errorf("Valid import should not produce errors, got: %v", errors)
	}

	// Verify module is registered in symbol table
	if !symbolTable.IsModuleImported("std") {
		t.Error("Module 'std' should be imported in symbol table")
	}

	// Test invalid import
	vclCode = `vcl 4.0;
import nonexistent;`

	program = parseVCL(t, vclCode)
	errors = validator.Validate(program)

	if len(errors) == 0 {
		t.Error("Invalid import should produce errors")
	}
}

func TestValidateFunctionCall(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Import module first
	vclCode := `vcl 4.0;
import std;

sub vcl_recv {
    set req.http.upper = std.toupper("hello");
}`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	if len(errors) != 0 {
		t.Errorf("Valid function call should not produce errors, got: %v", errors)
	}

	// Test function call without import
	vclCode = `vcl 4.0;

sub vcl_recv {
    set req.http.upper = std.toupper("hello");
}`

	program = parseVCL(t, vclCode)
	errors = validator.Validate(program)

	if len(errors) == 0 {
		t.Error("Function call without import should produce errors")
	}

	// Test non-existent function
	vclCode = `vcl 4.0;
import std;

sub vcl_recv {
    set req.http.result = std.nonexistent("hello");
}`

	program = parseVCL(t, vclCode)
	errors = validator.Validate(program)

	if len(errors) == 0 {
		t.Error("Non-existent function call should produce errors")
	}
}

func TestValidateObjectInstantiation(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test valid object instantiation
	vclCode := `vcl 4.0;
import directors;

sub vcl_init {
    new cluster = directors.round_robin();
}`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	if len(errors) != 0 {
		t.Errorf("Valid object instantiation should not produce errors, got: %v", errors)
	}

	// Test object instantiation without import
	vclCode = `vcl 4.0;

sub vcl_init {
    new cluster = directors.round_robin();
}`

	program = parseVCL(t, vclCode)
	errors = validator.Validate(program)

	if len(errors) == 0 {
		t.Error("Object instantiation without import should produce errors")
	}

	// Test non-existent object
	vclCode = `vcl 4.0;
import directors;

sub vcl_init {
    new cluster = directors.nonexistent();
}`

	program = parseVCL(t, vclCode)
	errors = validator.Validate(program)

	if len(errors) == 0 {
		t.Error("Non-existent object instantiation should produce errors")
	}
}

func TestValidateMethodCall(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test valid method call
	vclCode := `vcl 4.0;
import directors;

backend web1 {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_init {
    new cluster = directors.round_robin();
    cluster.add_backend(web1);
}

sub vcl_recv {
    set req.backend_hint = cluster.backend();
}`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	if len(errors) != 0 {
		t.Errorf("Valid method call should not produce errors, got: %v", errors)
	}
}

func TestValidateComplexVCL(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test complex VCL with multiple VMODs
	vclCode := `vcl 4.0;

import std;
import directors;

backend web1 {
    .host = "127.0.0.1";
    .port = "8080";
}

backend web2 {
    .host = "127.0.0.1";
    .port = "8081";
}

sub vcl_init {
    new cluster = directors.round_robin();
    cluster.add_backend(web1);
    cluster.add_backend(web2);

    new hash_cluster = directors.hash();
    hash_cluster.add_backend(web1, 1.0);
    hash_cluster.add_backend(web2, 2.0);
}

sub vcl_recv {
    if (std.file_exists("/maintenance")) {
        return (synth(503, "Maintenance"));
    }

    std.log("Processing request for " + req.url);

    if (req.url ~ "^/hash/") {
        set req.backend_hint = hash_cluster.backend(req.url);
    } else {
        set req.backend_hint = cluster.backend();
    }

    set req.http.x-random = std.random(1.0, 100.0);
    set req.http.x-upper = std.toupper(req.http.host);
}`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	if len(errors) != 0 {
		t.Errorf("Complex valid VCL should not produce errors, got: %v", errors)
	}
}

func TestValidateWithErrors(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test VCL with multiple errors
	vclCode := `vcl 4.0;

import std;
import nonexistent;

sub vcl_recv {
    # Missing import for directors
    new cluster = directors.round_robin();

    # Non-existent function
    set req.http.result = std.nonexistent("test");

    # Valid function call
    set req.http.upper = std.toupper("hello");

    # Function call on non-imported module
    set req.http.other = other.function("test");
}`

	program := parseVCL(t, vclCode)
	errors := validator.Validate(program)

	// Should have multiple errors
	if len(errors) < 3 {
		t.Errorf("Expected at least 3 errors, got %d: %v", len(errors), errors)
	}

	// Check that we have errors for:
	// 1. import nonexistent
	// 2. directors not imported
	// 3. std.nonexistent function
	// 4. other module not imported

	errorStrings := strings.Join(errors, " ")

	if !strings.Contains(errorStrings, "nonexistent") {
		t.Error("Should have error about nonexistent module")
	}

	if !strings.Contains(errorStrings, "directors") {
		t.Error("Should have error about directors module")
	}
}

func TestInferExpressionType(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	tests := []struct {
		name     string
		expr     ast.Expression
		expected string
	}{
		{
			name:     "string literal",
			expr:     &ast.StringLiteral{Value: "hello"},
			expected: "STRING",
		},
		{
			name:     "integer literal",
			expr:     &ast.IntegerLiteral{Value: 42},
			expected: "INT",
		},
		{
			name:     "float literal",
			expr:     &ast.FloatLiteral{Value: 3.14},
			expected: "REAL",
		},
		{
			name:     "boolean literal",
			expr:     &ast.BooleanLiteral{Value: true},
			expected: "BOOL",
		},
		{
			name:     "identifier",
			expr:     &ast.Identifier{Name: "req.method"},
			expected: "STRING", // Default assumption
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			vccType := validator.inferExpressionType(test.expr)
			if string(vccType) != test.expected {
				t.Errorf("Expected type %s, got %s", test.expected, vccType)
			}
		})
	}
}

func TestTypeConversion(t *testing.T) {
	registry := setupTestRegistry(t)
	symbolTable := types.NewSymbolTable()
	validator := NewVMODValidator(registry, symbolTable)

	// Test VCC to Symbol type conversion
	vccTests := map[string]types.Type{
		"STRING":      types.String,
		"INT":         types.Int,
		"REAL":        types.Real,
		"BOOL":        types.Bool,
		"BACKEND":     types.Backend,
		"HEADER":      types.Header,
		"DURATION":    types.Duration,
		"BYTES":       types.Bytes,
		"IP":          types.IP,
		"TIME":        types.Time,
		"VOID":        types.Void,
		"STRING_LIST": types.String, // Maps to string
	}

	for vccTypeStr, expectedSymbolType := range vccTests {
		vccType := vcc.VCCType(vccTypeStr)
		symbolType := validator.convertVCCTypeToSymbolType(vccType)
		if symbolType != expectedSymbolType {
			t.Errorf("VCC type %s: expected symbol type %s, got %s",
				vccTypeStr, expectedSymbolType, symbolType)
		}
	}

	// Test Symbol to VCC type conversion
	symbolTests := map[types.Type]string{
		types.String:   "STRING",
		types.Int:      "INT",
		types.Real:     "REAL",
		types.Bool:     "BOOL",
		types.Backend:  "BACKEND",
		types.Header:   "HEADER",
		types.Duration: "DURATION",
		types.Bytes:    "BYTES",
		types.IP:       "IP",
		types.Time:     "TIME",
		types.Void:     "VOID",
	}

	for symbolType, expectedVCCTypeStr := range symbolTests {
		vccType := validator.convertSymbolTypeToVCCType(symbolType)
		if string(vccType) != expectedVCCTypeStr {
			t.Errorf("Symbol type %s: expected VCC type %s, got %s",
				symbolType, expectedVCCTypeStr, vccType)
		}
	}
}
