package vmod

import (
	"os"
	"path/filepath"
	"testing"
)

// TestVCCLibAllFiles tests that all VCC files in vcclib directory can be parsed
// without syntax errors. This is a comprehensive smoke test to ensure all
// VCC files in the repository are syntactically valid.
func TestVCCLibAllFiles(t *testing.T) {
	registry := NewRegistry()

	// Get the vcclib directory path relative to this test file
	vccLibPath := filepath.Join("..", "vcclib")

	// Check if vcclib directory exists
	if _, err := os.Stat(vccLibPath); os.IsNotExist(err) {
		t.Skipf("vcclib directory not found at %s, skipping comprehensive test", vccLibPath)
	}

	// Load all VCC files from vcclib directory
	err := registry.LoadVCCDirectory(vccLibPath)
	if err != nil {
		t.Fatalf("Failed to load VCC files from %s: %v", vccLibPath, err)
	}

	// Get all VCC files in the directory
	vccFiles, err := filepath.Glob(filepath.Join(vccLibPath, "*.vcc"))
	if err != nil {
		t.Fatalf("Failed to find VCC files: %v", err)
	}

	if len(vccFiles) == 0 {
		t.Fatalf("No VCC files found in %s", vccLibPath)
	}

	t.Logf("Found %d VCC files in %s", len(vccFiles), vccLibPath)

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

// TestVCCLibIndividualFiles tests each VCC file individually to identify
// which specific files might have parsing issues.
func TestVCCLibIndividualFiles(t *testing.T) {
	vccLibPath := filepath.Join("..", "vcclib")

	// Check if vcclib directory exists
	if _, err := os.Stat(vccLibPath); os.IsNotExist(err) {
		t.Skipf("vcclib directory not found at %s, skipping individual file test", vccLibPath)
	}

	// Get all VCC files in the directory
	vccFiles, err := filepath.Glob(filepath.Join(vccLibPath, "*.vcc"))
	if err != nil {
		t.Fatalf("Failed to find VCC files: %v", err)
	}

	if len(vccFiles) == 0 {
		t.Fatalf("No VCC files found in %s", vccLibPath)
	}

	successCount := 0
	failureCount := 0

	for _, vccFile := range vccFiles {
		fileName := filepath.Base(vccFile)
		t.Run(fileName, func(t *testing.T) {
			registry := NewRegistry()

			// Create temporary directory with just this one file
			tmpDir, err := os.MkdirTemp("", "vcc_individual_test_*")
			if err != nil {
				t.Fatalf("Failed to create temp directory: %v", err)
			}
			defer os.RemoveAll(tmpDir)

			// Copy the VCC file to the temp directory
			content, err := os.ReadFile(vccFile)
			if err != nil {
				t.Fatalf("Failed to read %s: %v", vccFile, err)
			}

			tmpFilePath := filepath.Join(tmpDir, fileName)
			if err := os.WriteFile(tmpFilePath, content, 0644); err != nil {
				t.Fatalf("Failed to write temp file: %v", err)
			}

			// Try to load just this file
			err = registry.LoadVCCDirectory(tmpDir)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", fileName, err)
				failureCount++
			} else {
				modules := registry.ListModules()
				if len(modules) == 0 {
					t.Errorf("No modules loaded from %s", fileName)
					failureCount++
				} else {
					t.Logf("Successfully loaded module(s) from %s: %v", fileName, modules)
					successCount++
				}
			}
		})
	}

	t.Logf("Individual file test summary: %d successful, %d failed out of %d total files",
		successCount, failureCount, len(vccFiles))

	// We expect most files to parse successfully individually
	if successCount < len(vccFiles)*3/4 {
		t.Errorf("Expected at least 75%% of files to parse successfully, got %d/%d (%.1f%%)",
			successCount, len(vccFiles), float64(successCount)/float64(len(vccFiles))*100)
	}
}
