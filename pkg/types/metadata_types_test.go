package types

import (
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/metadata"
)

func TestMetadataTypeSystem_LoadTypes(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	mts := NewMetadataTypeSystem(loader)
	err = mts.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	// Verify some basic types are loaded
	allTypes, err := mts.GetAllTypes()
	if err != nil {
		t.Fatalf("Failed to get all types: %v", err)
	}

	expectedTypes := []string{"STRING", "INT", "BOOL", "IP", "BACKEND", "ACL", "HEADER", "TIME", "DURATION"}
	for _, typeName := range expectedTypes {
		if _, exists := allTypes[typeName]; !exists {
			t.Errorf("Expected type %s to be loaded", typeName)
		}
	}

	t.Logf("Loaded %d types", len(allTypes))
}

func TestMetadataTypeSystem_GetType(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	mts := NewMetadataTypeSystem(loader)

	// Test valid type
	stringType, err := mts.GetType("STRING")
	if err != nil {
		t.Fatalf("Failed to get STRING type: %v", err)
	}

	if stringType.String() != "STRING" {
		t.Errorf("Expected type name to be STRING, got %s", stringType.String())
	}

	// Test invalid type
	_, err = mts.GetType("INVALID_TYPE")
	if err == nil {
		t.Error("Expected error for invalid type, but got none")
	}

	// Test internal type (should be rejected)
	_, err = mts.GetType("STRINGS")
	if err == nil {
		t.Error("Expected error for internal type STRINGS, but got none")
	}
}

func TestMetadataTypeSystem_IsValidType(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	mts := NewMetadataTypeSystem(loader)

	tests := []struct {
		typeName string
		expected bool
	}{
		{"STRING", true},
		{"INT", true},
		{"BOOL", true},
		{"INVALID_TYPE", false},
		{"STRINGS", false}, // Internal type
	}

	for _, test := range tests {
		result := mts.IsValidType(test.typeName)
		if result != test.expected {
			t.Errorf("IsValidType(%s) = %v, expected %v", test.typeName, result, test.expected)
		}
	}
}

func TestMetadataTypeSystem_GetCType(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	mts := NewMetadataTypeSystem(loader)

	// Test getting C type mapping
	cType, err := mts.GetCType("STRING")
	if err != nil {
		t.Fatalf("Failed to get C type for STRING: %v", err)
	}

	if cType != "const char *" {
		t.Errorf("Expected C type 'const char *' for STRING, got '%s'", cType)
	}

	// Test invalid type
	_, err = mts.GetCType("INVALID_TYPE")
	if err == nil {
		t.Error("Expected error for invalid type, but got none")
	}
}

func TestInitializeMetadataTypes(t *testing.T) {
	// Reset global state
	DefaultMetadataTypeSystem = nil

	err := InitializeMetadataTypes()
	if err != nil {
		t.Fatalf("Failed to initialize metadata types: %v", err)
	}

	if DefaultMetadataTypeSystem == nil {
		t.Error("Expected DefaultMetadataTypeSystem to be initialized")
	}

	// Test getting a type through the global function
	stringType, err := GetMetadataType("STRING")
	if err != nil {
		t.Fatalf("Failed to get STRING type through global function: %v", err)
	}

	if stringType.String() != "STRING" {
		t.Errorf("Expected type name to be STRING, got %s", stringType.String())
	}
}

func TestInitializeWithMetadata(t *testing.T) {
	// Reset global state
	DefaultMetadataTypeSystem = nil
	MetadataString = nil

	err := InitializeWithMetadata()
	if err != nil {
		t.Fatalf("Failed to initialize with metadata: %v", err)
	}

	// Check that common types are initialized
	if MetadataString == nil {
		t.Error("Expected MetadataString to be initialized")
	}

	if MetadataString.String() != "STRING" {
		t.Errorf("Expected MetadataString to be STRING, got %s", MetadataString.String())
	}

	if MetadataInt == nil {
		t.Error("Expected MetadataInt to be initialized")
	}

	if MetadataBool == nil {
		t.Error("Expected MetadataBool to be initialized")
	}
}
