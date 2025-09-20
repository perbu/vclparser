package vclparser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varnish/vclparser/analyzer"
	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/vmod"
)

// Integration test for VMOD validation functionality
func TestVMODValidationIntegration(t *testing.T) {
	// Setup test VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create std.vcc
	stdVCC := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING_LIST s)
$Function STRING tolower(STRING_LIST s)
$Function VOID log(STRING_LIST s)
$Function REAL random(REAL lo, REAL hi)
$Function BOOL file_exists(STRING path)
$Function STRING fileread(PRIV_CALL, STRING path)`

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(stdVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	// Create directors.vcc
	directorsVCC := `$Module directors 3 "Directors module"
$ABI strict

$Object round_robin()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()

$Object hash()
$Method VOID .add_backend(BACKEND backend, REAL weight)
$Method BACKEND .backend(STRING key)`

	directorsFile := filepath.Join(tmpDir, "directors.vcc")
	err = os.WriteFile(directorsFile, []byte(directorsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write directors.vcc: %v", err)
	}

	// Create a custom registry and load our test VCC files
	registry := vmod.NewRegistry()
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	// Test cases
	testCases := []struct {
		name           string
		vclCode        string
		expectErrors   bool
		expectedErrors []string
	}{
		{
			name: "Valid VCL with VMOD usage",
			vclCode: `vcl 4.0;

import std;
import directors;

backend web1 {
    .host = "127.0.0.1";
    .port = "8080";
}

backend web2 {
    .host = "127.0.0.1";
    .port = "8081";
}

sub vcl_init {
    new cluster = directors.round_robin();
    cluster.add_backend(web1);
    cluster.add_backend(web2);
}

sub vcl_recv {
    std.log("Processing request: " + req.url);
    set req.http.x-upper = std.toupper(req.http.host);
    set req.backend_hint = cluster.backend();
}`,
			expectErrors: false,
		},
		{
			name: "Missing import",
			vclCode: `vcl 4.0;

sub vcl_recv {
    set req.http.upper = std.toupper("hello");
}`,
			expectErrors:   true,
			expectedErrors: []string{"not imported"},
		},
		{
			name: "Non-existent module",
			vclCode: `vcl 4.0;

import nonexistent;`,
			expectErrors:   true,
			expectedErrors: []string{"not available"},
		},
		{
			name: "Non-existent function",
			vclCode: `vcl 4.0;

import std;

sub vcl_recv {
    set req.http.result = std.nonexistent("test");
}`,
			expectErrors:   true,
			expectedErrors: []string{"not found"},
		},
		{
			name: "Object without import",
			vclCode: `vcl 4.0;

sub vcl_init {
    new cluster = directors.round_robin();
}`,
			expectErrors:   true,
			expectedErrors: []string{"not imported"},
		},
		{
			name: "Mixed valid and invalid",
			vclCode: `vcl 4.0;

import std;

sub vcl_recv {
    # Valid call
    set req.http.upper = std.toupper("hello");

    # Invalid call
    set req.http.result = std.nonexistent("test");

    # Invalid module
    set req.http.other = other.function("test");
}`,
			expectErrors:   true,
			expectedErrors: []string{"nonexistent", "not imported"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Parse with VMOD validation using our custom registry
			program, validationErrors, err := analyzer.ParseWithCustomVMODValidation(tc.vclCode, "test.vcl", registry)

			// Check parsing succeeded
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			if program == nil {
				t.Fatal("Program should not be nil")
			}

			// Check validation results
			if tc.expectErrors {
				if len(validationErrors) == 0 {
					t.Error("Expected validation errors but got none")
				} else {
					// Check that expected error patterns are present
					errorText := strings.Join(validationErrors, " ")
					for _, expectedError := range tc.expectedErrors {
						if !strings.Contains(strings.ToLower(errorText), strings.ToLower(expectedError)) {
							t.Errorf("Expected error containing '%s' but got: %v", expectedError, validationErrors)
						}
					}
				}
			} else {
				if len(validationErrors) > 0 {
					t.Errorf("Expected no validation errors but got: %v", validationErrors)
				}
			}
		})
	}
}

func TestVMODRegistryStats(t *testing.T) {
	// Setup test VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_stats_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create multiple VCC files
	vccFiles := map[string]string{
		"std.vcc": `$Module std 3 "Standard library"
$Function STRING toupper(STRING_LIST s)
$Function VOID log(STRING_LIST s)`,
		"directors.vcc": `$Module directors 3 "Directors module"
$Object round_robin()
$Method VOID .add_backend(BACKEND backend)`,
		"blob.vcc": `$Module blob 3 "Blob module"
$Function STRING encode(ENUM {BASE64, HEX} encoding, BLOB blob)
$Object decoder()
$Event vmod_event`,
	}

	for filename, content := range vccFiles {
		filepath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filepath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filename, err)
		}
	}

	// Create registry and load files
	registry := vmod.NewRegistry()
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	// Test stats
	stats := registry.GetModuleStats()

	if len(stats) != 3 {
		t.Errorf("Expected 3 modules in stats, got %d", len(stats))
	}

	// Check std module stats
	stdStats, exists := stats["std"]
	if !exists {
		t.Error("std module should exist in stats")
	} else {
		if stdStats.FunctionCount != 2 {
			t.Errorf("std: expected 2 functions, got %d", stdStats.FunctionCount)
		}
		if stdStats.ObjectCount != 0 {
			t.Errorf("std: expected 0 objects, got %d", stdStats.ObjectCount)
		}
	}

	// Check directors module stats
	directorsStats, exists := stats["directors"]
	if !exists {
		t.Error("directors module should exist in stats")
	} else {
		if directorsStats.FunctionCount != 0 {
			t.Errorf("directors: expected 0 functions, got %d", directorsStats.FunctionCount)
		}
		if directorsStats.ObjectCount != 1 {
			t.Errorf("directors: expected 1 object, got %d", directorsStats.ObjectCount)
		}
	}

	// Check blob module stats
	blobStats, exists := stats["blob"]
	if !exists {
		t.Error("blob module should exist in stats")
	} else {
		if blobStats.FunctionCount != 1 {
			t.Errorf("blob: expected 1 function, got %d", blobStats.FunctionCount)
		}
		if blobStats.ObjectCount != 1 {
			t.Errorf("blob: expected 1 object, got %d", blobStats.ObjectCount)
		}
		if blobStats.EventCount != 1 {
			t.Errorf("blob: expected 1 event, got %d", blobStats.EventCount)
		}
	}

	// Test builtin modules
	builtins := registry.GetBuiltinModules()
	expectedBuiltins := []string{"std", "directors"}

	if len(builtins) != len(expectedBuiltins) {
		t.Errorf("Expected %d builtin modules, got %d", len(expectedBuiltins), len(builtins))
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
			t.Errorf("Expected builtin module '%s' not found in %v", expected, builtins)
		}
	}
}

func TestVMODValidationWithRealVCL(t *testing.T) {
	// This test uses a more realistic VCL configuration
	vclCode := `vcl 4.0;

import std;
import directors;

backend web1 {
    .host = "192.168.1.10";
    .port = "80";
    .probe = {
        .url = "/health";
        .interval = 5s;
        .timeout = 1s;
        .window = 5;
        .threshold = 3;
    };
}

backend web2 {
    .host = "192.168.1.11";
    .port = "80";
    .probe = {
        .url = "/health";
        .interval = 5s;
        .timeout = 1s;
        .window = 5;
        .threshold = 3;
    };
}

acl purge {
    "localhost";
    "127.0.0.1";
    "::1";
}

sub vcl_init {
    new cluster = directors.round_robin();
    cluster.add_backend(web1);
    cluster.add_backend(web2);
}

sub vcl_recv {
    # Log incoming request
    std.log("Request: " + req.method + " " + req.url + " from " + client.ip);

    # Handle purge requests
    if (req.method == "PURGE") {
        if (!client.ip ~ purge) {
            return (synth(405, "Not allowed"));
        }
        return (purge);
    }

    # Only handle GET/HEAD requests
    if (req.method != "GET" && req.method != "HEAD") {
        return (pass);
    }

    # Set backend
    set req.backend_hint = cluster.backend();

    # Add some headers
    set req.http.X-Forwarded-For = client.ip;
    set req.http.X-Varnish-Upper = std.toupper(req.http.host);

    # Random header for testing
    set req.http.X-Random = std.random(1.0, 1000.0);
}

sub vcl_backend_response {
    # Log backend response
    std.log("Backend response: " + beresp.status + " for " + bereq.url);

    # Cache for 1 hour by default
    set beresp.ttl = 1h;
}

sub vcl_deliver {
    # Add response headers
    set resp.http.X-Served-By = "Varnish";
    if (obj.hits > 0) {
        set resp.http.X-Cache = "HIT";
    } else {
        set resp.http.X-Cache = "MISS";
    }
}`

	// Setup minimal VCC files for this test
	tmpDir, err := os.MkdirTemp("", "vcc_real_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	stdVCC := `$Module std 3 "Standard library"
$Function STRING toupper(STRING_LIST s)
$Function VOID log(STRING_LIST s)
$Function REAL random(REAL lo, REAL hi)`

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(stdVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	directorsVCC := `$Module directors 3 "Directors module"
$Object round_robin()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()`

	directorsFile := filepath.Join(tmpDir, "directors.vcc")
	err = os.WriteFile(directorsFile, []byte(directorsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write directors.vcc: %v", err)
	}

	// Create registry and load files
	registry := vmod.NewRegistry()
	err = registry.LoadVCCDirectory(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	// Override the default registry for this test
	oldRegistry := vmod.DefaultRegistry
	vmod.DefaultRegistry = registry
	defer func() { vmod.DefaultRegistry = oldRegistry }()

	// Parse with VMOD validation using our custom registry
	program, validationErrors, err := analyzer.ParseWithCustomVMODValidation(vclCode, "realistic.vcl", registry)

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if program == nil {
		t.Fatal("Program should not be nil")
	}

	// Should have no validation errors for this realistic VCL
	if len(validationErrors) > 0 {
		t.Errorf("Realistic VCL should not have validation errors, got: %v", validationErrors)
	}

	// Verify the program structure
	if program.VCLVersion == nil {
		t.Error("VCL version should be present")
	}

	if len(program.Declarations) == 0 {
		t.Error("Program should have declarations")
	}

	// Count different types of declarations
	imports := 0
	backends := 0
	subroutines := 0
	acls := 0

	for _, decl := range program.Declarations {
		switch decl.(type) {
		case *ast.ImportDecl:
			imports++
		case *ast.BackendDecl:
			backends++
		case *ast.SubDecl:
			subroutines++
		case *ast.ACLDecl:
			acls++
		}
	}

	if imports != 2 {
		t.Errorf("Expected 2 imports, got %d", imports)
	}

	if backends != 2 {
		t.Errorf("Expected 2 backends, got %d", backends)
	}

	if subroutines < 4 {
		t.Errorf("Expected at least 4 subroutines, got %d", subroutines)
	}

	if acls != 1 {
		t.Errorf("Expected 1 ACL, got %d", acls)
	}
}
