package metadata

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestMetadataLoader_LoadFromFile(t *testing.T) {
	loader := New()

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
	loader := New()
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
	loader := New()
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

func TestNormalizeDynamicVariable(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Valid HTTP header patterns
		{"req.http.user-agent", "req.http."},
		{"bereq.http.authorization", "bereq.http."},
		{"beresp.http.content-type", "beresp.http."},
		{"resp.http.cache-control", "resp.http."},
		{"obj.http.etag", "obj.http."},
		{"req_top.http.host", "req_top.http."},

		// Edge cases for HTTP headers
		{"req.http.", "req.http."},
		{"req.http..", "req.http."},
		{"req.http.header.with.dots", "req.http."},

		// Malformed HTTP patterns
		{"req.http", ""},
		{"req.httpfoo.bar", ""},
		{"req.http.foo.http.bar", ""}, // Multiple .http. should return empty

		// Storage patterns (currently returns empty)
		{"storage.malloc.free_space", ""},
		{"storage.default.used_space", ""},
		{"storage.foo", ""},
		{"storage.", ""},

		// Non-matching patterns
		{"req.url", ""},
		{"client.ip", ""},
		{"random.variable", ""},
		{"", ""},
	}

	for _, test := range tests {
		result := normalizeDynamicVariable(test.input)
		if result != test.expected {
			t.Errorf("normalizeDynamicVariable(%q) = %q, expected %q",
				test.input, result, test.expected)
		}
	}
}

func TestVCLVariable_ContextResolution(t *testing.T) {
	methods := map[string]VCLMethod{
		"recv":              {Context: "C", AllowedReturns: []string{"hash"}},
		"vcl_recv":          {Context: "C", AllowedReturns: []string{"hash"}},
		"backend_fetch":     {Context: "B", AllowedReturns: []string{"fetch"}},
		"vcl_backend_fetch": {Context: "B", AllowedReturns: []string{"fetch"}},
		"housekeeping":      {Context: "H", AllowedReturns: []string{"ok"}},
	}

	tests := []struct {
		name       string
		variable   VCLVariable
		method     string
		accessType string
		expected   bool
	}{
		{
			name: "all permission works for any method",
			variable: VCLVariable{
				ReadableFrom: []string{"all"},
			},
			method:     "recv",
			accessType: "read",
			expected:   true,
		},
		{
			name: "client permission works for client context",
			variable: VCLVariable{
				ReadableFrom: []string{"client"},
			},
			method:     "recv",
			accessType: "read",
			expected:   true,
		},
		{
			name: "client permission fails for backend context",
			variable: VCLVariable{
				ReadableFrom: []string{"client"},
			},
			method:     "backend_fetch",
			accessType: "read",
			expected:   false,
		},
		{
			name: "backend permission works for backend context",
			variable: VCLVariable{
				WritableFrom: []string{"backend"},
			},
			method:     "backend_fetch",
			accessType: "write",
			expected:   true,
		},
		{
			name: "both permission works for client context",
			variable: VCLVariable{
				UnsetableFrom: []string{"both"},
			},
			method:     "recv",
			accessType: "unset",
			expected:   true,
		},
		{
			name: "both permission works for backend context",
			variable: VCLVariable{
				ReadableFrom: []string{"both"},
			},
			method:     "backend_fetch",
			accessType: "read",
			expected:   true,
		},
		{
			name: "both permission fails for housekeeping context",
			variable: VCLVariable{
				ReadableFrom: []string{"both"},
			},
			method:     "housekeeping",
			accessType: "read",
			expected:   false,
		},
		{
			name: "direct method name match",
			variable: VCLVariable{
				ReadableFrom: []string{"recv"},
			},
			method:     "recv",
			accessType: "read",
			expected:   true,
		},
		{
			name: "vcl_ prefix method match",
			variable: VCLVariable{
				ReadableFrom: []string{"vcl_recv"},
			},
			method:     "recv",
			accessType: "read",
			expected:   true,
		},
		{
			name: "method without vcl_ prefix matches",
			variable: VCLVariable{
				ReadableFrom: []string{"vcl_recv"},
			},
			method:     "recv", // This should match because vcl_recv permission allows recv method
			accessType: "read",
			expected:   true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var result bool
			switch test.accessType {
			case "read":
				result = test.variable.IsReadableInMethod(test.method, methods)
			case "write":
				result = test.variable.IsWritableInMethod(test.method, methods)
			case "unset":
				result = test.variable.IsUnsetableInMethod(test.method, methods)
			}

			if result != test.expected {
				t.Errorf("Expected %v, got %v for %s access to method %s",
					test.expected, result, test.accessType, test.method)
			}
		})
	}
}

func TestMetadataLoader_ConcurrentAccess(t *testing.T) {
	loader := New()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup

	// Load default metadata in one goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := loader.LoadDefault(); err != nil {
			t.Errorf("LoadDefault failed: %v", err)
		}
	}()

	// Concurrently try to access metadata
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// Try different access patterns
				switch j % 4 {
				case 0:
					_, err := loader.GetMetadata()
					if err != nil && err.Error() != "metadata not loaded - call LoadFromFile or LoadDefault first" {
						t.Errorf("Goroutine %d: GetMetadata failed: %v", id, err)
					}
				case 1:
					_, err := loader.GetMethods()
					if err != nil && err.Error() != "metadata not loaded - call LoadFromFile or LoadDefault first" {
						t.Errorf("Goroutine %d: GetMethods failed: %v", id, err)
					}
				case 2:
					_, err := loader.GetVariables()
					if err != nil && err.Error() != "metadata not loaded - call LoadFromFile or LoadDefault first" {
						t.Errorf("Goroutine %d: GetVariables failed: %v", id, err)
					}
				case 3:
					err := loader.ValidateReturnAction("recv", "hash")
					if err != nil && err.Error() != "metadata not loaded - call LoadFromFile or LoadDefault first" {
						t.Errorf("Goroutine %d: ValidateReturnAction failed: %v", id, err)
					}
				}

				// Add small delay to increase chance of race conditions
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify final state is valid
	metadata, err := loader.GetMetadata()
	if err != nil {
		t.Errorf("Final metadata access failed: %v", err)
	}

	if metadata == nil {
		t.Error("Expected metadata to be loaded after concurrent operations")
	}
}

func TestMetadataLoader_ErrorConditions(t *testing.T) {
	t.Run("access before loading", func(t *testing.T) {
		loader := New()

		_, err := loader.GetMetadata()
		if err == nil || err.Error() != "metadata not loaded - call LoadFromFile or LoadDefault first" {
			t.Errorf("Expected specific error for unloaded metadata, got: %v", err)
		}

		_, err = loader.GetMethods()
		if err == nil {
			t.Error("Expected error when accessing methods before loading")
		}

		_, err = loader.GetVariables()
		if err == nil {
			t.Error("Expected error when accessing variables before loading")
		}
	})

	t.Run("invalid file path", func(t *testing.T) {
		loader := New()

		err := loader.LoadFromFile("/nonexistent/path/metadata.json")
		if err == nil {
			t.Error("Expected error for nonexistent file")
		}

		if err != nil && !filepath.IsAbs("/nonexistent/path/metadata.json") {
			t.Error("Error handling should work with absolute paths")
		}
	})

	t.Run("invalid JSON content", func(t *testing.T) {
		// This would require creating a temporary file with invalid JSON
		// For now, we'll test that the error handling exists
		loader := New()

		// Try to load from a directory (will fail)
		err := loader.LoadFromFile(".")
		if err == nil {
			t.Error("Expected error when trying to load directory as JSON")
		}
	})

	t.Run("unknown method validation", func(t *testing.T) {
		loader := New()
		err := loader.LoadDefault()
		if err != nil {
			t.Fatalf("LoadDefault failed: %v", err)
		}

		err = loader.ValidateReturnAction("nonexistent_method", "hash")
		if err == nil {
			t.Error("Expected error for unknown method")
		}

		expectedMsg := "unknown VCL method: nonexistent_method"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})

	t.Run("invalid access type", func(t *testing.T) {
		loader := New()
		err := loader.LoadDefault()
		if err != nil {
			t.Fatalf("LoadDefault failed: %v", err)
		}

		err = loader.ValidateVariableAccess("req.url", "recv", "invalid_access")
		if err == nil {
			t.Error("Expected error for invalid access type")
		}

		expectedMsg := "invalid access type: invalid_access (must be read, write, or unset)"
		if err.Error() != expectedMsg {
			t.Errorf("Expected error message %q, got %q", expectedMsg, err.Error())
		}
	})
}

func TestVCLVariable_VersionEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		variable VCLVariable
		version  int
		expected bool
	}{
		{
			name: "exact version low boundary",
			variable: VCLVariable{
				VersionLow:  40,
				VersionHigh: 41,
			},
			version:  40,
			expected: true,
		},
		{
			name: "exact version high boundary",
			variable: VCLVariable{
				VersionLow:  40,
				VersionHigh: 41,
			},
			version:  41,
			expected: true,
		},
		{
			name: "invalid version range (high < low)",
			variable: VCLVariable{
				VersionLow:  41,
				VersionHigh: 40,
			},
			version:  40,
			expected: false,
		},
		{
			name: "zero versions",
			variable: VCLVariable{
				VersionLow:  0,
				VersionHigh: 0,
			},
			version:  0,
			expected: true,
		},
		{
			name: "negative version check",
			variable: VCLVariable{
				VersionLow:  40,
				VersionHigh: 41,
			},
			version:  -1,
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.variable.IsAvailableInVersion(test.version)
			if result != test.expected {
				t.Errorf("Expected %v for version %d with range [%d, %d], got %v",
					test.expected, test.version, test.variable.VersionLow, test.variable.VersionHigh, result)
			}
		})
	}
}

func TestStorageVariablePatterns(t *testing.T) {
	loader := New()
	err := loader.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault failed: %v", err)
	}

	storageVars, err := loader.GetStorageVariables()
	if err != nil {
		t.Fatalf("GetStorageVariables failed: %v", err)
	}

	if len(storageVars) == 0 {
		t.Skip("No storage variables in metadata to test")
	}

	// Test that storage variables are properly loaded
	foundFreeSpace := false
	foundUsedSpace := false
	foundHappy := false

	for _, sv := range storageVars {
		switch sv.Name {
		case "free_space":
			foundFreeSpace = true
			if sv.Type != "BYTES" {
				t.Errorf("Expected free_space to be BYTES type, got %s", sv.Type)
			}
		case "used_space":
			foundUsedSpace = true
			if sv.Type != "BYTES" {
				t.Errorf("Expected used_space to be BYTES type, got %s", sv.Type)
			}
		case "happy":
			foundHappy = true
			if sv.Type != "BOOL" {
				t.Errorf("Expected happy to be BOOL type, got %s", sv.Type)
			}
		}
	}

	if !foundFreeSpace {
		t.Error("Expected to find free_space storage variable")
	}
	if !foundUsedSpace {
		t.Error("Expected to find used_space storage variable")
	}
	if !foundHappy {
		t.Error("Expected to find happy storage variable")
	}

	// Test storage variable validation patterns
	tests := []struct {
		name       string
		variable   string
		shouldFind bool
	}{
		{"concrete storage variable", "storage.malloc.free_space", false}, // Currently not implemented
		{"generic storage pattern", "storage.default.used_space", false},  // Currently not implemented
		{"invalid storage variable", "storage.nonexistent.property", false},
		{"malformed storage pattern", "storage.malloc", false},
		{"empty storage name", "storage..free_space", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := loader.ValidateVariableAccess(test.variable, "recv", "read")
			hasError := err != nil

			if test.shouldFind && hasError {
				t.Errorf("Expected %s to be valid, got error: %v", test.variable, err)
			} else if !test.shouldFind && !hasError {
				t.Errorf("Expected %s to be invalid, but validation passed", test.variable)
			}
		})
	}
}

func TestLoadDefaultVsLoadFromFile(t *testing.T) {
	t.Run("LoadDefault then LoadFromFile", func(t *testing.T) {
		loader := New()

		// Load default first
		err := loader.LoadDefault()
		if err != nil {
			t.Fatalf("LoadDefault failed: %v", err)
		}

		defaultMetadata, err := loader.GetMetadata()
		if err != nil {
			t.Fatalf("GetMetadata after LoadDefault failed: %v", err)
		}

		// Try to load from file (should replace default)
		projectRoot := "../../"
		metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

		err = loader.LoadFromFile(metadataPath)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		fileMetadata, err := loader.GetMetadata()
		if err != nil {
			t.Fatalf("GetMetadata after LoadFromFile failed: %v", err)
		}

		// Both should have similar structure (assuming metadata.json matches embedded)
		if len(defaultMetadata.VCLMethods) == 0 || len(fileMetadata.VCLMethods) == 0 {
			t.Error("Both metadata sources should have VCL methods")
		}
	})

	t.Run("LoadFromFile then LoadDefault", func(t *testing.T) {
		loader := New()

		// Load from file first
		projectRoot := "../../"
		metadataPath := filepath.Join(projectRoot, "metadata", "metadata.json")

		err := loader.LoadFromFile(metadataPath)
		if err != nil {
			t.Fatalf("LoadFromFile failed: %v", err)
		}

		fileMetadata, err := loader.GetMetadata()
		if err != nil {
			t.Fatalf("GetMetadata after LoadFromFile failed: %v", err)
		}

		// Load default (should replace file metadata)
		err = loader.LoadDefault()
		if err != nil {
			t.Fatalf("LoadDefault failed: %v", err)
		}

		defaultMetadata, err := loader.GetMetadata()
		if err != nil {
			t.Fatalf("GetMetadata after LoadDefault failed: %v", err)
		}

		// Verify metadata was replaced
		if len(fileMetadata.VCLMethods) == 0 || len(defaultMetadata.VCLMethods) == 0 {
			t.Error("Both metadata sources should have VCL methods")
		}
	})

	t.Run("multiple LoadDefault calls", func(t *testing.T) {
		loader := New()

		// Multiple calls should be safe
		for i := 0; i < 3; i++ {
			err := loader.LoadDefault()
			if err != nil {
				t.Fatalf("LoadDefault call %d failed: %v", i+1, err)
			}

			metadata, err := loader.GetMetadata()
			if err != nil {
				t.Fatalf("GetMetadata after LoadDefault call %d failed: %v", i+1, err)
			}

			if metadata == nil {
				t.Fatalf("Expected metadata to be non-nil after LoadDefault call %d", i+1)
			}
		}
	})

	t.Run("global convenience functions", func(t *testing.T) {
		// Test the global DefaultLoader convenience functions
		err := LoadDefault()
		if err != nil {
			t.Fatalf("Global LoadDefault failed: %v", err)
		}

		metadata, err := GetMetadata()
		if err != nil {
			t.Fatalf("Global GetMetadata failed: %v", err)
		}

		if metadata == nil {
			t.Error("Expected global metadata to be non-nil")
		}

		if len(metadata.VCLMethods) == 0 {
			t.Error("Expected global metadata to have VCL methods")
		}
	})
}
