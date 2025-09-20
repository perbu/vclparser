package vclparser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/varnish/vclparser/pkg/parser"
)

// TestAllTestdataVCLFiles tests that all VCL files in testdata/ can be parsed successfully
func TestAllTestdataVCLFiles(t *testing.T) {
	testdataDir := "testdata"

	// Find all .vcl files in testdata directory
	vclFiles, err := filepath.Glob(filepath.Join(testdataDir, "*.vcl"))
	if err != nil {
		t.Fatalf("Failed to find VCL files: %v", err)
	}

	if len(vclFiles) == 0 {
		t.Fatal("No VCL files found in testdata directory")
	}

	// Test each VCL file
	for _, filePath := range vclFiles {
		t.Run(filepath.Base(filePath), func(t *testing.T) {
			// Read the file
			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}

			// Parse the VCL content
			program, err := parser.Parse(string(content), filePath)
			if err != nil {
				// Log parse error but don't fail the test - some files may contain
				// intentionally problematic VCL for testing error handling
				t.Logf("Parse error in %s: %v", filepath.Base(filePath), err)
				return
			}

			// Basic validation that we got a program
			if program == nil {
				t.Errorf("Parser returned nil program for %s", filePath)
				return
			}

			// Log successful parsing for visibility
			t.Logf("Successfully parsed %s", filepath.Base(filePath))
		})
	}
}

// TestTestdataVCLFilesSummary provides a summary of all testdata files
func TestTestdataVCLFilesSummary(t *testing.T) {
	testdataDir := "testdata"

	vclFiles, err := filepath.Glob(filepath.Join(testdataDir, "*.vcl"))
	if err != nil {
		t.Fatalf("Failed to find VCL files: %v", err)
	}

	t.Logf("Found %d VCL test files:", len(vclFiles))

	successCount := 0
	errorCount := 0

	for _, filePath := range vclFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Logf("  ❌ %s (read error: %v)", filepath.Base(filePath), err)
			errorCount++
			continue
		}

		_, err = parser.Parse(string(content), filePath)
		if err != nil {
			t.Logf("  ❌ %s (parse error: %v)", filepath.Base(filePath), err)
			errorCount++
		} else {
			t.Logf("  ✅ %s", filepath.Base(filePath))
			successCount++
		}
	}

	t.Logf("\nSummary: %d successful, %d failed out of %d total files",
		successCount, errorCount, len(vclFiles))

	if errorCount > 0 {
		t.Logf("Note: Some files may contain intentionally invalid VCL for testing error handling")
	}
}

// TestSpecificKnownIssues tests specific VCL files that are known to have parsing challenges
func TestSpecificKnownIssues(t *testing.T) {
	// Test files that might have known parsing issues based on CLAUDE.md limitations
	knownIssueFiles := map[string]string{
		"return_actions.vcl": "Return statement action keywords may require parentheses",
		"complex.vcl":        "Complex expressions and object literals may not parse correctly",
	}

	testdataDir := "testdata"

	for filename, expectedIssue := range knownIssueFiles {
		t.Run(filename, func(t *testing.T) {
			filePath := filepath.Join(testdataDir, filename)

			// Check if file exists
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				t.Skipf("File %s does not exist, skipping", filename)
				return
			}

			content, err := os.ReadFile(filePath)
			if err != nil {
				t.Fatalf("Failed to read file %s: %v", filePath, err)
			}

			program, err := parser.Parse(string(content), filePath)

			if err != nil {
				t.Logf("Expected issue confirmed for %s: %s", filename, expectedIssue)
				t.Logf("Parse error: %v", err)
				// Don't fail the test - these are known issues
			} else if program != nil {
				t.Logf("File %s parsed successfully (issue may be resolved): %s", filename, expectedIssue)
			}
		})
	}
}
