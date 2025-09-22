package vclparser_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
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
				t.Fatalf("Parse error in %s: %v", filepath.Base(filePath), err)
			}
			// Basic validation that we got a program
			if program == nil {
				t.Fatalf("Parser returned nil program for %s", filePath)
			}
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

	for _, filePath := range vclFiles {
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file %s: %v", filePath, err)
		}

		if _, err := parser.Parse(string(content), filePath); err != nil {
			t.Errorf("Parse error in %s: %v", filepath.Base(filePath), err)
		}
	}
}
