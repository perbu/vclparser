package vmod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/vcc"
)

func TestRegistryBasicOperations(t *testing.T) {
	registry := NewEmptyRegistry()

	// Test empty registry
	if len(registry.ListModules()) != 0 {
		t.Error("New registry should be empty")
	}

	// Create a test VCC content
	vccContent := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING_LIST s)
$Function VOID log(STRING_LIST s)

$Object test_object()
$Method VOID .method1()
$Method STRING .method2(INT param)`

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "test_*.vcc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(vccContent)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Load the VCC file
	err = registry.LoadVCCFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load VCC file: %v", err)
	}

	// Test module exists
	modules := registry.ListModules()
	if len(modules) != 1 {
		t.Errorf("Expected 1 module, got %d", len(modules))
	}

	if modules[0] != "std" {
		t.Errorf("Expected module 'std', got '%s'", modules[0])
	}

	// Test GetModule
	module, exists := registry.GetModule("std")
	if !exists {
		t.Error("Module 'std' should exist")
	}
	if module == nil {
		t.Fatal("Module should not be nil when exists is true")
	}

	if module.Name != "std" {
		t.Errorf("Expected module name 'std', got '%s'", module.Name)
	}

	// Test GetFunction
	function, err := registry.GetFunction("std", "toupper")
	if err != nil {
		t.Errorf("Failed to get function: %v", err)
	}
	if function == nil {
		t.Fatal("Function should not be nil when no error returned")
	}

	if function.Name != "toupper" {
		t.Errorf("Expected function name 'toupper', got '%s'", function.Name)
	}

	// Test GetObject
	object, err := registry.GetObject("std", "test_object")
	if err != nil {
		t.Errorf("Failed to get object: %v", err)
	}
	if object == nil {
		t.Fatal("Object should not be nil when no error returned")
	}

	if object.Name != "test_object" {
		t.Errorf("Expected object name 'test_object', got '%s'", object.Name)
	}

	// Test GetMethod
	method, err := registry.GetMethod("std", "test_object", "method1")
	if err != nil {
		t.Errorf("Failed to get method: %v", err)
	}
	if method == nil {
		t.Fatal("Method should not be nil when no error returned")
	}

	if method.Name != "method1" {
		t.Errorf("Expected method name 'method1', got '%s'", method.Name)
	}
}

func TestRegistryValidation(t *testing.T) {
	registry := NewEmptyRegistry()

	// Create a test VCC content
	vccContent := `$Module std 3 "Standard library"

$Function STRING toupper(STRING_LIST s)
$Function REAL random(REAL lo, REAL hi)

$Object round_robin()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()`

	// Load from string
	tmpFile, err := os.CreateTemp("", "test_*.vcc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(vccContent)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	err = registry.LoadVCCFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load VCC file: %v", err)
	}

	// Test ValidateImport
	err = registry.ValidateImport("std")
	if err != nil {
		t.Errorf("Import validation should succeed: %v", err)
	}

	err = registry.ValidateImport("nonexistent")
	if err == nil {
		t.Error("Import validation should fail for non-existent module")
	}

	// Test ValidateFunctionCall
	err = registry.ValidateFunctionCall("std", "toupper", []vcc.VCCType{vcc.TypeString})
	if err != nil {
		t.Errorf("Function call validation should succeed: %v", err)
	}

	err = registry.ValidateFunctionCall("std", "toupper", []vcc.VCCType{vcc.TypeInt})
	if err == nil {
		t.Error("Function call validation should fail for wrong argument type")
	}

	err = registry.ValidateFunctionCall("std", "random", []vcc.VCCType{vcc.TypeReal, vcc.TypeReal})
	if err != nil {
		t.Errorf("Function call validation should succeed: %v", err)
	}

	err = registry.ValidateFunctionCall("std", "random", []vcc.VCCType{vcc.TypeReal})
	if err == nil {
		t.Error("Function call validation should fail for insufficient arguments")
	}

	// Test ValidateObjectConstruction
	err = registry.ValidateObjectConstruction("std", "round_robin", []vcc.VCCType{})
	if err != nil {
		t.Errorf("Object construction validation should succeed: %v", err)
	}

	// Test ValidateMethodCall
	err = registry.ValidateMethodCall("std", "round_robin", "add_backend", []vcc.VCCType{vcc.TypeBackend})
	if err != nil {
		t.Errorf("Method call validation should succeed: %v", err)
	}

	err = registry.ValidateMethodCall("std", "round_robin", "add_backend", []vcc.VCCType{vcc.TypeString})
	if err == nil {
		t.Error("Method call validation should fail for wrong argument type")
	}
}

func TestRegistryLoadDirectory(t *testing.T) {
	registry := NewEmptyRegistry()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create multiple VCC files
	vccFiles := map[string]string{
		"std.vcc": `$Module std 3 "Standard library"
$Function STRING toupper(STRING_LIST s)`,
		"directors.vcc": `$Module directors 3 "Directors module"
$Object round_robin()`,
		"not_vcc.txt": "This is not a VCC file",
	}

	for filename, content := range vccFiles {
		filepath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filepath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filename, err)
		}
	}

	// Load VCC files individually
	for filename := range vccFiles {
		if strings.HasSuffix(strings.ToLower(filename), ".vcc") {
			filePath := filepath.Join(tmpDir, filename)
			err := registry.LoadVCCFile(filePath)
			if err != nil {
				t.Fatalf("Failed to load VCC file %s: %v", filename, err)
			}
		}
	}

	// Check loaded modules
	modules := registry.ListModules()
	if len(modules) != 2 {
		t.Errorf("Expected 2 modules, got %d", len(modules))
	}

	// Check specific modules
	if !registry.ModuleExists("std") {
		t.Error("Module 'std' should exist")
	}

	if !registry.ModuleExists("directors") {
		t.Error("Module 'directors' should exist")
	}

	if registry.ModuleExists("nonexistent") {
		t.Error("Module 'nonexistent' should not exist")
	}
}

func TestRegistryStats(t *testing.T) {
	registry := NewEmptyRegistry()

	// Create test module
	vccContent := `$Module test 1 "Test module"
$Function STRING func1()
$Function VOID func2()
$Object obj1()
$Object obj2()
$Event event1`

	tmpFile, err := os.CreateTemp("", "test_*.vcc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(vccContent)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	err = registry.LoadVCCFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load VCC file: %v", err)
	}

	// Test stats
	stats := registry.GetModuleStats()
	if len(stats) != 1 {
		t.Errorf("Expected 1 module in stats, got %d", len(stats))
	}

	testStats, exists := stats["test"]
	if !exists {
		t.Error("Module 'test' should exist in stats")
	}

	if testStats.Name != "test" {
		t.Errorf("Expected module name 'test', got '%s'", testStats.Name)
	}

	if testStats.Version != 1 {
		t.Errorf("Expected version 1, got %d", testStats.Version)
	}

	if testStats.FunctionCount != 2 {
		t.Errorf("Expected 2 functions, got %d", testStats.FunctionCount)
	}

	if testStats.ObjectCount != 2 {
		t.Errorf("Expected 2 objects, got %d", testStats.ObjectCount)
	}

	if testStats.EventCount != 1 {
		t.Errorf("Expected 1 event, got %d", testStats.EventCount)
	}
}

func TestRegistryClear(t *testing.T) {
	registry := NewEmptyRegistry()

	// Load a module
	vccContent := `$Module std 3 "Standard library"
$Function STRING toupper(STRING_LIST s)`

	tmpFile, err := os.CreateTemp("", "test_*.vcc")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer func() {
		if err := os.Remove(tmpFile.Name()); err != nil {
			t.Logf("Failed to remove temp file: %v", err)
		}
	}()

	_, err = tmpFile.WriteString(vccContent)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	err = registry.LoadVCCFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load VCC file: %v", err)
	}

	// Verify module is loaded
	if len(registry.ListModules()) != 1 {
		t.Error("Registry should have 1 module")
	}

	// Clear registry
	registry.Clear()

	// Verify registry is empty
	if len(registry.ListModules()) != 0 {
		t.Error("Registry should be empty after clear")
	}

	if registry.ModuleExists("std") {
		t.Error("Module 'std' should not exist after clear")
	}
}

func TestBuiltinModules(t *testing.T) {
	registry := NewEmptyRegistry()

	// Load standard modules
	vccStd := `$Module std 3 "Standard library"
$Function STRING toupper(STRING_LIST s)`

	vccDirectors := `$Module directors 3 "Directors module"
$Object round_robin()`

	// Create temp files
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(vccStd), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	directorsFile := filepath.Join(tmpDir, "directors.vcc")
	err = os.WriteFile(directorsFile, []byte(vccDirectors), 0644)
	if err != nil {
		t.Fatalf("Failed to write directors.vcc: %v", err)
	}

	// Load VCC files individually
	err = registry.LoadVCCFile(stdFile)
	if err != nil {
		t.Fatalf("Failed to load std.vcc: %v", err)
	}

	err = registry.LoadVCCFile(directorsFile)
	if err != nil {
		t.Fatalf("Failed to load directors.vcc: %v", err)
	}

	// Test builtin modules
	builtins := registry.GetBuiltinModules()
	expectedBuiltins := []string{"std", "directors"}

	if len(builtins) != len(expectedBuiltins) {
		t.Errorf("Expected %d builtin modules, got %d", len(expectedBuiltins), len(builtins))
	}

	for _, expected := range expectedBuiltins {
		found := false
		for _, builtin := range builtins {
			if builtin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected builtin module '%s' not found", expected)
		}
	}
}
