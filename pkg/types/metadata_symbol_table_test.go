package types

import (
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/metadata"
)

func TestMetadataSymbolTable_LoadBuiltinSymbols(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	err = typeSystem.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	mst := NewMetadataSymbolTable(loader, typeSystem)
	err = mst.LoadBuiltinSymbols()
	if err != nil {
		t.Fatalf("Failed to load builtin symbols: %v", err)
	}

	// Test that some key variables are loaded
	testVars := []string{"client.ip", "req.method", "req.url", "beresp.status"}
	for _, varName := range testVars {
		symbol := mst.Lookup(varName)
		if symbol == nil {
			t.Errorf("Expected variable %s to be loaded", varName)
		} else {
			t.Logf("Variable %s: Type=%s, Readable=%v, Writable=%v",
				varName, symbol.Type.String(), symbol.Readable, symbol.Writable)
		}
	}
}

func TestMetadataSymbolTable_ValidateVariableAccess(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	err = typeSystem.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	mst := NewMetadataSymbolTable(loader, typeSystem)

	tests := []struct {
		variable   string
		method     string
		accessType string
		expected   bool
	}{
		{"client.ip", "recv", "read", true},
		{"client.ip", "recv", "write", false},
		{"req.url", "recv", "read", true},
		{"req.url", "recv", "write", true},
		{"bereq.url", "deliver", "write", false},
		{"bereq.url", "backend_fetch", "write", true},
	}

	for _, test := range tests {
		err := mst.ValidateVariableAccess(test.variable, test.method, test.accessType)
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

func TestMetadataSymbolTable_ValidateReturnAction(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	err = typeSystem.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	mst := NewMetadataSymbolTable(loader, typeSystem)

	tests := []struct {
		method   string
		action   string
		expected bool
	}{
		{"recv", "hash", true},
		{"recv", "pass", true},
		{"recv", "lookup", false},
		{"hash", "lookup", true},
		{"hash", "pass", false},
		{"deliver", "deliver", true},
		{"deliver", "fetch", false},
	}

	for _, test := range tests {
		err := mst.ValidateReturnAction(test.method, test.action)
		hasError := err != nil

		if test.expected && hasError {
			t.Errorf("Expected %s+%s to be valid, got error: %v", test.method, test.action, err)
		} else if !test.expected && !hasError {
			t.Errorf("Expected %s+%s to be invalid, but validation passed", test.method, test.action)
		}
	}
}

func TestMetadataSymbolTable_LookupWithAccess(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	err = typeSystem.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	mst := NewMetadataSymbolTable(loader, typeSystem)
	err = mst.LoadBuiltinSymbols()
	if err != nil {
		t.Fatalf("Failed to load builtin symbols: %v", err)
	}

	// Test valid access
	symbol, err := mst.LookupWithAccess("client.ip", "recv", "read")
	if err != nil {
		t.Errorf("Expected client.ip read in recv to be valid, got error: %v", err)
	}
	if symbol == nil {
		t.Error("Expected to get a symbol for client.ip")
	}

	// Test invalid access
	_, err = mst.LookupWithAccess("client.ip", "recv", "write")
	if err == nil {
		t.Error("Expected client.ip write in recv to be invalid")
	}

	// Test dynamic variable (HTTP header)
	symbol, err = mst.LookupWithAccess("req.http.user-agent", "recv", "read")
	if err != nil {
		t.Errorf("Expected req.http.user-agent read in recv to be valid, got error: %v", err)
	}
	if symbol == nil {
		t.Error("Expected to get a symbol for dynamic variable")
	}
}

func TestMetadataSymbolTable_HandleDynamicVariable(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	err = typeSystem.LoadTypes()
	if err != nil {
		t.Fatalf("Failed to load types: %v", err)
	}

	mst := NewMetadataSymbolTable(loader, typeSystem)

	tests := []struct {
		variable   string
		method     string
		accessType string
		expected   bool
	}{
		{"req.http.user-agent", "recv", "read", true},
		{"req.http.authorization", "recv", "write", true},
		{"beresp.http.content-type", "backend_response", "read", true},
		{"resp.http.server", "deliver", "write", true},
		{"storage.malloc.free_space", "recv", "read", true},
		{"storage.malloc.free_space", "recv", "write", false},
		{"invalid.pattern", "recv", "read", false},
	}

	for _, test := range tests {
		err := mst.handleDynamicVariable(test.variable, test.method, test.accessType)
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

func TestMetadataSymbolTable_GetMethodContext(t *testing.T) {
	loader := metadata.NewMetadataLoader()
	projectRoot := "../../"
	metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

	err := loader.LoadFromFile(metadataPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	typeSystem := NewMetadataTypeSystem(loader)
	mst := NewMetadataSymbolTable(loader, typeSystem)

	tests := []struct {
		method   string
		expected metadata.ContextType
	}{
		{"recv", metadata.ClientContext},
		{"deliver", metadata.ClientContext},
		{"backend_fetch", metadata.BackendContext},
		{"backend_response", metadata.BackendContext},
		{"init", metadata.HousekeepingContext},
		{"fini", metadata.HousekeepingContext},
	}

	for _, test := range tests {
		context, err := mst.GetMethodContext(test.method)
		if err != nil {
			t.Errorf("Failed to get context for method %s: %v", test.method, err)
			continue
		}

		if context != test.expected {
			t.Errorf("Expected context %s for method %s, got %s",
				test.expected, test.method, context)
		}
	}
}

func TestCreateDefault(t *testing.T) {
	mst, err := CreateDefault()
	if err != nil {
		t.Fatalf("Failed to create default metadata symbol table: %v", err)
	}

	// Test that basic variables are available
	symbol := mst.Lookup("req.url")
	if symbol == nil {
		t.Error("Expected req.url to be available in default symbol table")
	}

	// Test validation works
	err = mst.ValidateVariableAccess("req.url", "recv", "read")
	if err != nil {
		t.Errorf("Expected req.url read in recv to be valid: %v", err)
	}
}
