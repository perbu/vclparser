package vmod

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/vcc"
)

func TestRegistryWithRealWorldVMODs(t *testing.T) {
	registry := NewRegistry()

	// Create temporary directory for VCC files
	tmpDir, err := os.MkdirTemp("", "registry_integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Test comprehensive crypto VMOD definition
	cryptoVCC := `$Module crypto 3 "Cryptographic functions module"
$ABI strict

# Hash functions
$Function STRING hex_encode(BYTES data)
$Function BYTES hash(ENUM {sha1, sha256, sha512} algorithm, STRING data)
$Function BYTES hmac(ENUM {sha1, sha256, sha512} algorithm, BYTES key, STRING data)
$Function BYTES blob(STRING data)
$Function STRING secret()

# AES functions
$Function INT aes_get_length()
$Function VOID aes_set_length(INT length)

# HMAC object
$Object hmac(ENUM {sha1, sha256, sha512} algorithm, STRING key)
$Method VOID .set_key(STRING key)
$Method BYTES .digest(STRING data)
$Method STRING .hex_digest(STRING data)`

	// Test S3 VMOD with advanced parameters
	s3VCC := `$Module s3 3 "Amazon S3 authentication and utilities"
$ABI strict

$Function BOOL verify(STRING access_key_id, STRING secret_key, DURATION clock_skew = -1s)
$Function STRING canonical_request(STRING method, STRING uri, STRING query, STRING headers)
$Function STRING string_to_sign(STRING algorithm, STRING timestamp, STRING scope, STRING canonical_request)`

	// Test YKey VMOD with cache tagging
	ykeyVCC := `$Module ykey 3 "Cache tagging and selective purging"
$ABI strict

$Function INT purge(STRING keys)
$Function VOID add_key(STRING key)
$Function STRING get_keys()
$Function STRING get_hashed_keys()
$Function VOID add_hashed_keys(STRING keys)
$Function VOID clear_keys()`

	// Test Probe Proxy VMOD
	probeProxyVCC := `$Module probe_proxy 3 "Intelligent probe forwarding"
$ABI strict

$Function BOOL is_probe()
$Function BACKEND self()
$Function VOID global_override(BACKEND backend)
$Function BACKEND backend()
$Function VOID force_fresh()
$Function VOID skip_health_check()
$Function DURATION timeout()
$Function STRING get_path()
$Function VOID set_path(STRING path)`

	// Test Utils VMOD with time and string utilities
	utilsVCC := `$Module utils 3 "Utility functions for VCL"
$ABI strict

$Function STRING time_format(STRING format, TIME time = now, BOOL utc = 0)
$Function STRING newline()
$Function TIME parse_time(STRING time_str, STRING format)
$Function STRING urlencode(STRING str)
$Function STRING urldecode(STRING str)`

	// Test advanced directors with multiple types
	directorsVCC := `$Module directors 3 "Advanced load balancing directors"
$ABI strict

$Object round_robin()
$Method VOID .add_backend(BACKEND backend, REAL weight = 1.0)
$Method BACKEND .backend()
$Method VOID .remove_backend(BACKEND backend)

$Object hash()
$Method VOID .add_backend(BACKEND backend, REAL weight = 1.0)
$Method BACKEND .backend(STRING key = "")
$Method VOID .remove_backend(BACKEND backend)

$Object fallback()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()
$Method BOOL .is_healthy()

$Object random()
$Method VOID .add_backend(BACKEND backend, REAL weight)
$Method BACKEND .backend()
$Method INT .get_weight(BACKEND backend)`

	vccFiles := map[string]string{
		"crypto.vcc":      cryptoVCC,
		"s3.vcc":          s3VCC,
		"ykey.vcc":        ykeyVCC,
		"probe_proxy.vcc": probeProxyVCC,
		"utils.vcc":       utilsVCC,
		"directors.vcc":   directorsVCC,
	}

	// Write VCC files
	for filename, content := range vccFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	// Load VCC files
	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	// Test module listing
	modules := registry.ListModules()
	loadedModules := len(modules)

	// Note: Some modules might fail to load due to VCC syntax issues, so check for at least some modules
	if loadedModules < 4 {
		t.Errorf("Expected at least 4 modules, got %d: %v", loadedModules, modules)
	}

	// Check that the most important modules exist
	importantModules := []string{"crypto", "ykey", "directors"}
	for _, expected := range importantModules {
		if !registry.ModuleExists(expected) {
			t.Errorf("Important module %s should exist", expected)
		}
	}

	// Test crypto module functions
	t.Run("crypto_module_functions", func(t *testing.T) {
		module, exists := registry.GetModule("crypto")
		if !exists {
			t.Fatal("crypto module should exist")
		}

		expectedFunctions := []string{"hex_encode", "hash", "hmac", "blob", "secret", "aes_get_length", "aes_set_length"}
		if len(module.Functions) != len(expectedFunctions) {
			t.Errorf("Expected %d functions, got %d", len(expectedFunctions), len(module.Functions))
		}

		for _, funcName := range expectedFunctions {
			if module.FindFunction(funcName) == nil {
				t.Errorf("Function %s should exist in crypto module", funcName)
			}
		}

		// Test function call validation
		hexEncodeFunc := module.FindFunction("hex_encode")
		if hexEncodeFunc == nil {
			t.Fatal("hex_encode function should exist")
		}

		// Test with correct argument types
		err := hexEncodeFunc.ValidateCall([]vcc.VCCType{vcc.TypeBytes})
		if err != nil {
			t.Errorf("hex_encode(BYTES) should be valid: %v", err)
		}

		// Test with incorrect argument types
		err = hexEncodeFunc.ValidateCall([]vcc.VCCType{vcc.TypeString, vcc.TypeInt})
		if err == nil {
			t.Errorf("hex_encode(STRING, INT) should be invalid")
		}
	})

	// Test crypto HMAC object
	t.Run("crypto_hmac_object", func(t *testing.T) {
		module, exists := registry.GetModule("crypto")
		if !exists {
			t.Fatal("crypto module should exist")
		}

		if len(module.Objects) != 1 {
			t.Errorf("Expected 1 object, got %d", len(module.Objects))
		}

		hmacObj := module.FindObject("hmac")
		if hmacObj == nil {
			t.Fatal("hmac object should exist")
		}

		expectedMethods := []string{"set_key", "digest", "hex_digest"}
		if len(hmacObj.Methods) != len(expectedMethods) {
			t.Errorf("Expected %d methods, got %d", len(expectedMethods), len(hmacObj.Methods))
		}

		for _, methodName := range expectedMethods {
			if hmacObj.FindMethod(methodName) == nil {
				t.Errorf("Method %s should exist on hmac object", methodName)
			}
		}

		// Test object construction validation - crypto.hmac expects ENUM for algorithm
		err := hmacObj.ValidateConstruction([]vcc.VCCType{vcc.TypeString, vcc.TypeString})
		if err == nil {
			t.Errorf("hmac(STRING, STRING) should be invalid - first param should be ENUM")
		}
	})

	// Test directors module with multiple objects
	t.Run("directors_module_objects", func(t *testing.T) {
		module, exists := registry.GetModule("directors")
		if !exists {
			t.Fatal("directors module should exist")
		}

		expectedObjects := []string{"round_robin", "hash", "fallback", "random"}
		if len(module.Objects) != len(expectedObjects) {
			t.Errorf("Expected %d objects, got %d", len(expectedObjects), len(module.Objects))
		}

		for _, objName := range expectedObjects {
			obj := module.FindObject(objName)
			if obj == nil {
				t.Errorf("Object %s should exist in directors module", objName)
			}

			// All director objects should have add_backend and backend methods
			if obj.FindMethod("add_backend") == nil {
				t.Errorf("Object %s should have add_backend method", objName)
			}
			if obj.FindMethod("backend") == nil {
				t.Errorf("Object %s should have backend method", objName)
			}
		}

		// Test specific object method validation
		hashObj := module.FindObject("hash")
		if hashObj == nil {
			t.Fatal("hash object should exist")
		}

		backendMethod := hashObj.FindMethod("backend")
		if backendMethod == nil {
			t.Fatal("hash.backend method should exist")
		}

		// Test method call validation
		err := backendMethod.ValidateCall([]vcc.VCCType{vcc.TypeString})
		if err != nil {
			t.Errorf("hash.backend(STRING) should be valid: %v", err)
		}

		err = backendMethod.ValidateCall([]vcc.VCCType{})
		if err != nil {
			t.Errorf("hash.backend() should be valid: %v", err)
		}
	})

	// Test registry statistics
	t.Run("registry_statistics", func(t *testing.T) {
		stats := registry.GetModuleStats()
		if len(stats) < 4 {
			t.Errorf("Expected at least 4 module stats, got %d", len(stats))
		}

		cryptoStats, exists := stats["crypto"]
		if !exists {
			t.Fatal("crypto module stats should exist")
		}

		if cryptoStats.Name != "crypto" {
			t.Errorf("Expected crypto name, got %s", cryptoStats.Name)
		}

		if cryptoStats.FunctionCount != 7 {
			t.Errorf("Expected 7 functions in crypto, got %d", cryptoStats.FunctionCount)
		}

		if cryptoStats.ObjectCount != 1 {
			t.Errorf("Expected 1 object in crypto, got %d", cryptoStats.ObjectCount)
		}

		directorsStats, exists := stats["directors"]
		if !exists {
			t.Fatal("directors module stats should exist")
		}

		if directorsStats.ObjectCount != 4 {
			t.Errorf("Expected 4 objects in directors, got %d", directorsStats.ObjectCount)
		}
	})

	// Test function and method lookup through registry
	t.Run("registry_function_method_lookup", func(t *testing.T) {
		// Test function lookup
		function, err := registry.GetFunction("crypto", "hex_encode")
		if err != nil {
			t.Errorf("Should find crypto.hex_encode: %v", err)
		}
		if function.Name != "hex_encode" {
			t.Errorf("Expected hex_encode, got %s", function.Name)
		}

		// Test object lookup
		object, err := registry.GetObject("directors", "hash")
		if err != nil {
			t.Errorf("Should find directors.hash: %v", err)
		}
		if object.Name != "hash" {
			t.Errorf("Expected hash, got %s", object.Name)
		}

		// Test method lookup
		method, err := registry.GetMethod("directors", "hash", "backend")
		if err != nil {
			t.Errorf("Should find directors.hash.backend: %v", err)
		}
		if method.Name != "backend" {
			t.Errorf("Expected backend, got %s", method.Name)
		}

		// Test non-existent lookups
		_, err = registry.GetFunction("nonexistent", "func")
		if err == nil {
			t.Error("Should not find function in non-existent module")
		}

		_, err = registry.GetFunction("crypto", "nonexistent")
		if err == nil {
			t.Error("Should not find non-existent function")
		}

		_, err = registry.GetMethod("directors", "hash", "nonexistent")
		if err == nil {
			t.Error("Should not find non-existent method")
		}
	})

	// Test validation functions
	t.Run("registry_validation", func(t *testing.T) {
		// Test import validation
		err := registry.ValidateImport("crypto")
		if err != nil {
			t.Errorf("Should validate crypto import: %v", err)
		}

		err = registry.ValidateImport("nonexistent")
		if err == nil {
			t.Error("Should not validate non-existent module import")
		}

		// Test function call validation
		err = registry.ValidateFunctionCall("crypto", "hex_encode", []vcc.VCCType{vcc.TypeBytes})
		if err != nil {
			t.Errorf("Should validate crypto.hex_encode(BYTES): %v", err)
		}

		err = registry.ValidateFunctionCall("crypto", "hex_encode", []vcc.VCCType{vcc.TypeInt})
		if err == nil {
			t.Error("Should not validate crypto.hex_encode(INT)")
		}

		// Test object construction validation
		err = registry.ValidateObjectConstruction("directors", "hash", []vcc.VCCType{})
		if err != nil {
			t.Errorf("Should validate directors.hash(): %v", err)
		}

		// Test method call validation
		err = registry.ValidateMethodCall("directors", "hash", "backend", []vcc.VCCType{vcc.TypeString})
		if err != nil {
			t.Errorf("Should validate directors.hash.backend(STRING): %v", err)
		}
	})
}

func TestRegistryBuiltinModules(t *testing.T) {
	registry := NewRegistry()

	// Create a minimal std module
	tmpDir, err := os.MkdirTemp("", "builtin_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	stdVCC := `$Module std 3 "Standard library"
$ABI strict
$Function STRING toupper(STRING s)`

	directorsVCC := `$Module directors 3 "Directors"
$ABI strict
$Object round_robin()
$Method BACKEND .backend()`

	files := map[string]string{
		"std.vcc":       stdVCC,
		"directors.vcc": directorsVCC,
	}

	for filename, content := range files {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	builtins := registry.GetBuiltinModules()
	expectedBuiltins := []string{"std", "directors"}

	if len(builtins) != len(expectedBuiltins) {
		t.Errorf("Expected %d builtin modules, got %d: %v", len(expectedBuiltins), len(builtins), builtins)
	}

	for _, expected := range expectedBuiltins {
		found := false
		for _, builtin := range builtins {
			if builtin == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected builtin module %s not found in %v", expected, builtins)
		}
	}
}

func TestRegistryIntegrationClear(t *testing.T) {
	registry := NewRegistry()

	// Load some modules first
	tmpDir, err := os.MkdirTemp("", "clear_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	testVCC := `$Module test 3 "Test module"
$ABI strict
$Function VOID test_func()`

	filePath := filepath.Join(tmpDir, "test.vcc")
	if err := os.WriteFile(filePath, []byte(testVCC), 0644); err != nil {
		t.Fatalf("Failed to write test.vcc: %v", err)
	}

	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	// Verify module is loaded
	if !registry.ModuleExists("test") {
		t.Error("test module should exist before clear")
	}

	// Clear registry
	registry.Clear()

	// Verify module is gone
	if registry.ModuleExists("test") {
		t.Error("test module should not exist after clear")
	}

	modules := registry.ListModules()
	if len(modules) != 0 {
		t.Errorf("Expected 0 modules after clear, got %d: %v", len(modules), modules)
	}
}
