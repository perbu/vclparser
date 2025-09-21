package include

import (
	"os"
	"path/filepath"
)

// FileReader provides an interface for reading files, allowing for easier testing
// and alternative file sources
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// OSFileReader implements FileReader using the standard os package
type OSFileReader struct {
	basePath string
}

// NewOSFileReader creates a new OSFileReader with the given base path
func NewOSFileReader(basePath string) *OSFileReader {
	return &OSFileReader{basePath: basePath}
}

// ReadFile reads a file, resolving relative paths against the base path
func (r *OSFileReader) ReadFile(path string) ([]byte, error) {
	var fullPath string

	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(r.basePath, path)
	}

	return os.ReadFile(fullPath)
}

// MemoryFileReader implements FileReader using an in-memory map for testing
type MemoryFileReader struct {
	files map[string]string
}

// NewMemoryFileReader creates a new MemoryFileReader with the given file contents
func NewMemoryFileReader(files map[string]string) *MemoryFileReader {
	return &MemoryFileReader{files: files}
}

// ReadFile reads a file from memory
func (r *MemoryFileReader) ReadFile(path string) ([]byte, error) {
	content, exists := r.files[path]
	if !exists {
		return nil, os.ErrNotExist
	}
	return []byte(content), nil
}

// AddFile adds a file to the memory reader
func (r *MemoryFileReader) AddFile(path, content string) {
	r.files[path] = content
}
