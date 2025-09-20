package vclparser_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/varnish/vclparser/pkg/analyzer"
	"github.com/varnish/vclparser/pkg/lexer"
	"github.com/varnish/vclparser/pkg/parser"
	"github.com/varnish/vclparser/pkg/types"
	"github.com/varnish/vclparser/pkg/vmod"
)

// Test VMOD functionality with simplified examples that match current parser capabilities
func TestSimplifiedVMODIntegration(t *testing.T) {
	registry := vmod.NewRegistry()

	tmpDir, err := os.MkdirTemp("", "vmod_simplified_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create VCC files with known working syntax
	cryptoVCC := `$Module crypto 3 "Cryptographic functions"
$ABI strict

$Function STRING hex_encode(BYTES data)
$Function BYTES hash(ENUM {sha1, sha256, sha512} algorithm, STRING data)
$Function BYTES hmac(ENUM {sha1, sha256, sha512} algorithm, BYTES key, STRING data)
$Function STRING secret()
$Function INT aes_get_length()
$Function VOID aes_set_length(INT length)

$Object hmac_obj(ENUM {sha1, sha256, sha512} algorithm, STRING key)
$Method VOID .set_key(STRING key)
$Method BYTES .digest(STRING data)`

	ykeyVCC := `$Module ykey 3 "Cache tagging"
$ABI strict

$Function INT purge(STRING keys)
$Function VOID add_key(STRING key)
$Function STRING get_hashed_keys()
$Function VOID add_hashed_keys(STRING keys)`

	stdVCC := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING s)
$Function VOID log(STRING s)
$Function INT integer(STRING s, INT fallback)
$Function BOOL healthy(BACKEND backend)`

	directorsVCC := `$Module directors 3 "Load balancing directors"
$ABI strict

$Object round_robin()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()

$Object fallback()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()`

	vccFiles := map[string]string{
		"crypto.vcc":    cryptoVCC,
		"ykey.vcc":      ykeyVCC,
		"std.vcc":       stdVCC,
		"directors.vcc": directorsVCC,
	}

	for filename, content := range vccFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "basic_crypto_usage",
			vcl: `vcl 4.0;
import crypto;

sub vcl_recv {
    set req.http.secret = crypto.secret();
    set req.http.len = crypto.aes_get_length();
}`,
			expectErrors: false,
		},
		{
			name: "ykey_cache_tagging",
			vcl: `vcl 4.0;
import ykey;

sub vcl_recv {
    if (req.http.purge) {
        set req.http.npurge = ykey.purge(req.http.purge);
        return (pass);
    }
}

sub vcl_backend_response {
    ykey.add_key("tag1");
    ykey.add_key(beresp.http.cache_tag);
}`,
			expectErrors: false,
		},
		{
			name: "directors_object_usage",
			vcl: `vcl 4.0;
import directors;

sub vcl_init {
    new rr = directors.round_robin();
    rr.add_backend(default);

    new fb = directors.fallback();
    fb.add_backend(default);
}

sub vcl_recv {
    set req.backend_hint = rr.backend();
}`,
			expectErrors: false,
		},
		{
			name: "std_library_functions",
			vcl: `vcl 4.0;
import std;

sub vcl_recv {
    std.log("Processing request");
    set req.http.upper = std.toupper(req.http.host);
    set req.http.port = std.integer(req.http.X-Port, 80);
}`,
			expectErrors: false,
		},
		{
			name: "crypto_with_object",
			vcl: `vcl 4.0;
import crypto;

sub vcl_init {
    new hasher = crypto.hmac_obj("sha256", "secret");
}

sub vcl_recv {
    hasher.set_key("new-key");
    set req.http.digest = hasher.digest("data");
}`,
			expectErrors:  true,
			errorContains: []string{"expected ENUM"},
		},
		{
			name: "mixed_vmod_usage",
			vcl: `vcl 4.0;
import std;
import ykey;
import crypto;

sub vcl_recv {
    std.log("Request processing started");
    set req.http.secret = crypto.secret();
}

sub vcl_backend_response {
    ykey.add_key("cache-tag");
    crypto.aes_set_length(256);
}`,
			expectErrors: false,
		},
		{
			name: "error_missing_import",
			vcl: `vcl 4.0;

sub vcl_recv {
    set req.http.secret = crypto.secret();
}`,
			expectErrors:  true,
			errorContains: []string{"not imported"},
		},
		{
			name: "error_unknown_function",
			vcl: `vcl 4.0;
import crypto;

sub vcl_recv {
    set req.http.result = crypto.nonexistent_function("data");
}`,
			expectErrors:  true,
			errorContains: []string{"not found"},
		},
		{
			name: "error_object_without_import",
			vcl: `vcl 4.0;

sub vcl_init {
    new rr = directors.round_robin();
}`,
			expectErrors:  true,
			errorContains: []string{"not imported"},
		},
		{
			name: "error_wrong_argument_types",
			vcl: `vcl 4.0;
import crypto;

sub vcl_recv {
    set req.http.result = crypto.hex_encode("not-bytes");
}`,
			expectErrors:  true,
			errorContains: []string{"expected BYTES, got STRING"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.vcl, "test.vcl")
			p := parser.New(l, tt.vcl, "test.vcl")
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				if !tt.expectErrors {
					t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
				}
				return // Parsing failed as expected
			}

			symbolTable := types.NewSymbolTable()
			validator := analyzer.NewVMODValidator(registry, symbolTable)
			errors := validator.Validate(program)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}

			for _, expectedError := range tt.errorContains {
				found := false
				for _, err := range errors {
					if strings.Contains(err, expectedError) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' but not found in: %v", expectedError, errors)
				}
			}
		})
	}
}

// Test that the VMOD system correctly handles real-world patterns found in vmod-vcl.md
func TestVMODPatterns(t *testing.T) {
	registry := vmod.NewRegistry()

	tmpDir, err := os.MkdirTemp("", "vmod_patterns_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	}()

	// Create comprehensive VMODs that cover the patterns in vmod-vcl.md
	vhaVCC := `$Module vha 3 "Varnish High Availability"
$ABI strict

$Function VOID log(STRING message)`

	strVCC := `$Module str 3 "String manipulation"
$ABI strict

$Function BOOL contains(STRING haystack, STRING needle)`

	kvstoreVCC := `$Module kvstore 3 "Key-value store"
$ABI strict

$Object init()
$Method STRING .get(STRING key)
$Method VOID .set(STRING key, STRING value)`

	vccFiles := map[string]string{
		"vha.vcc":     vhaVCC,
		"str.vcc":     strVCC,
		"kvstore.vcc": kvstoreVCC,
	}

	for filename, content := range vccFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC directory: %v", err)
	}

	tests := []struct {
		name         string
		vcl          string
		expectErrors bool
	}{
		{
			name: "vha6_export_import_pattern",
			vcl: `vcl 4.0;
import vha;

sub vcl_deliver {
    if (req.method == "VHA_FETCH") {
        set resp.http.vha6-data = "exported-data";
        if (resp.http.vha6-data == "") {
            unset resp.http.vha6-data;
        }
    }
}

sub vcl_backend_response {
    if (bereq.method == "VHA_FETCH") {
        if (beresp.http.vha6-data) {
            vha.log("VHA_BROADCAST PEER: Data import");
        }
        unset beresp.http.vha6-data;
    }
}`,
			expectErrors: false,
		},
		{
			name: "kvstore_usage_pattern",
			vcl: `vcl 4.0;
import kvstore;
import str;

sub vcl_init {
    new opts = kvstore.init();
}

sub vcl_recv {
    if (str.contains(req.http.VPP-path, server.identity)) {
        return (synth(508));
    }

    if (opts.get("call_recv") != "true") {
        return (hash);
    }
}`,
			expectErrors: false,
		},
		{
			name: "initialization_pattern",
			vcl: `vcl 4.0;
import kvstore;

sub vcl_init {
    new settings = kvstore.init();
    settings.set("debug", "1");
    settings.set("timeout", "30");
}

sub vcl_recv {
    if (settings.get("debug") == "1") {
        return (synth(200, "Debug mode"));
    }
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := lexer.New(tt.vcl, "test.vcl")
			p := parser.New(l, tt.vcl, "test.vcl")
			program := p.ParseProgram()

			if len(p.Errors()) > 0 {
				if !tt.expectErrors {
					t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
				}
				return
			}

			symbolTable := types.NewSymbolTable()
			validator := analyzer.NewVMODValidator(registry, symbolTable)
			errors := validator.Validate(program)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}
		})
	}
}

// Test that the VMOD registry correctly loads from embedded VCC files
func TestVMODWithEmbeddedVCCLib(t *testing.T) {
	// The registry should already be loaded with embedded VCC files via init()
	// Test that we have modules loaded automatically

	// Test that we can load some common modules
	commonModules := []string{"std", "directors", "blob", "debug"}
	loadedCount := 0

	for _, moduleName := range commonModules {
		if vmod.DefaultRegistry.ModuleExists(moduleName) {
			loadedCount++
			t.Logf("Successfully loaded embedded module: %s", moduleName)
		}
	}

	if loadedCount == 0 {
		t.Error("No common modules were loaded from embedded VCC files")
	}

	// Test that we can get stats
	stats := vmod.DefaultRegistry.GetModuleStats()
	t.Logf("Loaded %d modules from embedded VCC files", len(stats))

	// Verify we have a reasonable number of modules
	if len(stats) < 5 {
		t.Errorf("Expected at least 5 embedded modules, got %d", len(stats))
	}

	// Print some module info for debugging
	for name, stat := range stats {
		t.Logf("Module %s: %d functions, %d objects", name, stat.FunctionCount, stat.ObjectCount)
		if name == "std" || name == "directors" {
			// Test validation with a real module
			err := vmod.DefaultRegistry.ValidateImport(name)
			if err != nil {
				t.Errorf("Should be able to validate import of %s: %v", name, err)
			}
		}
	}
}
