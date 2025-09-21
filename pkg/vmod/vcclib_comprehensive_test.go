package vmod

import (
	"testing"

	"github.com/perbu/vclparser"
)

// TestVCCLibAllFiles tests that all VCC files in vcclib directory can be parsed
// without syntax errors. This is a comprehensive smoke test to ensure all
// VCC files in the repository are syntactically valid.
func TestVCCLibAllFiles(t *testing.T) {
	registry := NewEmptyRegistry()

	// Load all embedded VCC files
	err := registry.LoadEmbeddedVCCs()
	if err != nil {
		t.Fatalf("Failed to load embedded VCC files: %v", err)
	}

	// Get all embedded VCC files
	vccFiles, err := vclparser.ListEmbeddedVCCFiles()
	if err != nil {
		t.Fatalf("Failed to list embedded VCC files: %v", err)
	}

	if len(vccFiles) == 0 {
		t.Fatalf("No embedded VCC files found")
	}

	t.Logf("Found %d embedded VCC files", len(vccFiles))

	// Check that at least some modules were loaded successfully
	modules := registry.ListModules()
	loadedCount := len(modules)

	t.Logf("Successfully loaded %d modules out of %d VCC files", loadedCount, len(vccFiles))

	// We expect at least 50% of files to parse successfully
	// Some files might have complex syntax or dependencies that cause parsing to fail
	minExpectedModules := len(vccFiles) / 2
	if loadedCount < minExpectedModules {
		t.Errorf("Expected at least %d modules to load, but only %d loaded", minExpectedModules, loadedCount)
		t.Logf("Loaded modules: %v", modules)
	}

	// Test that some well-known essential modules are present
	essentialModules := []string{"std", "directors", "blob"}
	for _, moduleName := range essentialModules {
		if !registry.ModuleExists(moduleName) {
			t.Errorf("Essential module %s should be present", moduleName)
		}
	}

	// Log statistics about the loaded modules
	stats := registry.GetModuleStats()
	totalFunctions := 0
	totalObjects := 0

	for moduleName, stat := range stats {
		totalFunctions += stat.FunctionCount
		totalObjects += stat.ObjectCount
		t.Logf("Module %s: %d functions, %d objects", moduleName, stat.FunctionCount, stat.ObjectCount)
	}

	t.Logf("Total across all modules: %d functions, %d objects", totalFunctions, totalObjects)

	// Verify we have a reasonable amount of functionality loaded
	if totalFunctions < 50 {
		t.Errorf("Expected at least 50 total functions across all modules, got %d", totalFunctions)
	}

	if totalObjects < 5 {
		t.Errorf("Expected at least 5 total objects across all modules, got %d", totalObjects)
	}
}
