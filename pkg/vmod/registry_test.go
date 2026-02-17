package vmod

import (
	"os"
	"path/filepath"
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

func TestRegistryLoadSOFile(t *testing.T) {
	registry := NewEmptyRegistry()

	stdPath := mustFindVMODFixture(t, "libvmod_std.so")
	if err := registry.LoadSOFile(stdPath); err != nil {
		t.Fatalf("Failed to load SO file: %v", err)
	}

	module, exists := registry.GetModule("std")
	if !exists {
		t.Fatalf("Expected std module to exist")
	}
	if module == nil {
		t.Fatalf("Expected std module to be non-nil")
	}

	function := module.FindFunction("toupper")
	if function == nil {
		t.Fatalf("Expected std.toupper function to exist")
	}
	if function.ReturnType != vcc.TypeString {
		t.Fatalf("Expected std.toupper return type STRING, got %s", function.ReturnType)
	}
}

func TestRegistryLoadSODirectory(t *testing.T) {
	registry := NewEmptyRegistry()

	vmodsDir := mustFindVMODFixtureDir(t)
	if err := registry.LoadSODirectory(vmodsDir); err != nil {
		t.Fatalf("Failed to load SO directory: %v", err)
	}

	if !registry.ModuleExists("std") {
		t.Fatalf("Expected std module to be loaded from SO directory")
	}
	if !registry.ModuleExists("directors") {
		t.Fatalf("Expected directors module to be loaded from SO directory")
	}
}

func mustFindVMODFixture(t *testing.T, filename string) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "vmods", filename),
		filepath.Join("vmods", filename),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	t.Skipf("VMOD fixture %s not found", filename)
	return ""
}

func mustFindVMODFixtureDir(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "..", "vmods"),
		"vmods",
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate
		}
	}

	t.Skip("VMOD fixtures directory not found")
	return ""
}
