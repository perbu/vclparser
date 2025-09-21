package metadata

import (
	"path/filepath"
	"testing"
)

func TestMetadataLoader_LoadFromFile(t *testing.T) {
	loader := NewMetadataLoader()

	// Get the test metadata file path relative to project root
	projectRoot := "../../" // From pkg/metadata/ to project root
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	metadata, err := loader.GetMetadata()
	if err != nil {
		t.Fatalf("Failed to get metadata: %v", err)
	}

	// Verify basic structure
	if len(metadata.VCLMethods) == 0 {
		t.Error("Expected VCL methods to be loaded")
	}

	if len(metadata.VCLVariables) == 0 {
		t.Error("Expected VCL variables to be loaded")
	}

	if len(metadata.VCLTypes) == 0 {
		t.Error("Expected VCL types to be loaded")
	}

	if len(metadata.VCLTokens) == 0 {
		t.Error("Expected VCL tokens to be loaded")
	}

	t.Logf("Loaded %d methods, %d variables, %d types, %d tokens",
		len(metadata.VCLMethods), len(metadata.VCLVariables),
		len(metadata.VCLTypes), len(metadata.VCLTokens))
}

func TestMetadataLoader_ValidateReturnAction(t *testing.T) {
	loader := NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	tests := []struct {
		method   string
		action   string
		expected bool
	}{
		{"recv", "hash", true},
		{"recv", "pass", true},
		{"recv", "lookup", false}, // lookup is only valid in hash
		{"hash", "lookup", true},
		{"hash", "pass", false}, // pass is not valid in hash
		{"deliver", "deliver", true},
		{"deliver", "fetch", false}, // fetch is not valid in deliver
	}

	for _, test := range tests {
		err := loader.ValidateReturnAction(test.method, test.action)
		hasError := err != nil

		if test.expected && hasError {
			t.Errorf("Expected %s+%s to be valid, got error: %v", test.method, test.action, err)
		} else if !test.expected && !hasError {
			t.Errorf("Expected %s+%s to be invalid, but validation passed", test.method, test.action)
		}
	}
}

func TestMetadataLoader_ValidateVariableAccess(t *testing.T) {
	loader := NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	// Test some basic variable access patterns
	tests := []struct {
		variable   string
		method     string
		accessType string
		expected   bool
	}{
		{"client.ip", "recv", "read", true},           // client.ip is readable in client methods
		{"client.ip", "recv", "write", false},         // client.ip is read-only
		{"req.url", "recv", "read", true},             // req.url is readable in recv
		{"req.url", "recv", "write", true},            // req.url is writable in recv
		{"bereq.url", "deliver", "write", false},      // bereq.url is not writable in client methods
		{"bereq.url", "backend_fetch", "write", true}, // bereq.url is writable in backend methods
	}

	for _, test := range tests {
		err := loader.ValidateVariableAccess(test.variable, test.method, test.accessType)
		hasError := err != nil

		if test.expected && hasError {
			t.Errorf("Expected %s.%s in %s to be valid, got error: %v",
				test.variable, test.accessType, test.method, err)
		} else if !test.expected && !hasError {
			t.Errorf("Expected %s.%s in %s to be invalid, but validation passed",
				test.variable, test.accessType, test.method)
		}
	}
}

func TestVCLMethod_IsValidReturnAction(t *testing.T) {
	method := VCLMethod{
		Context:        "C",
		AllowedReturns: []string{"hash", "pass", "pipe", "fail"},
	}

	if !method.IsValidReturnAction("hash") {
		t.Error("Expected 'hash' to be valid")
	}

	if method.IsValidReturnAction("lookup") {
		t.Error("Expected 'lookup' to be invalid")
	}
}

func TestVCLVariable_IsAvailableInVersion(t *testing.T) {
	variable := VCLVariable{
		VersionLow:  40,
		VersionHigh: 41,
	}

	if !variable.IsAvailableInVersion(40) {
		t.Error("Expected variable to be available in version 40")
	}

	if !variable.IsAvailableInVersion(41) {
		t.Error("Expected variable to be available in version 41")
	}

	if variable.IsAvailableInVersion(39) {
		t.Error("Expected variable to not be available in version 39")
	}

	if variable.IsAvailableInVersion(42) {
		t.Error("Expected variable to not be available in version 42")
	}
}
