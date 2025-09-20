package vclparser

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
)

//go:embed vcclib
var embeddedVCCFiles embed.FS

// GetEmbeddedVCCFiles returns the embedded filesystem containing all VCC files
func GetEmbeddedVCCFiles() embed.FS {
	return embeddedVCCFiles
}

// ListEmbeddedVCCFiles returns a list of all embedded VCC file paths
func ListEmbeddedVCCFiles() ([]string, error) {
	var vccFiles []string

	err := fs.WalkDir(embeddedVCCFiles, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".vcc") {
			vccFiles = append(vccFiles, path)
		}

		return nil
	})

	return vccFiles, err
}

// OpenEmbeddedVCCFile opens a specific embedded VCC file for reading
func OpenEmbeddedVCCFile(filename string) (io.ReadCloser, error) {
	// Handle both relative and full paths
	if !strings.HasPrefix(filename, "vcclib/") {
		filename = filepath.Join("vcclib", filename)
	}

	file, err := embeddedVCCFiles.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open embedded VCC file %s: %v", filename, err)
	}

	return file, nil
}

// GetEmbeddedVCCContent reads the entire content of an embedded VCC file
func GetEmbeddedVCCContent(filename string) ([]byte, error) {
	file, err := OpenEmbeddedVCCFile(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint: errcheck

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded VCC file %s: %v", filename, err)
	}

	return content, nil
}
