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

// setupRealWorldVMODs creates VCC files for VMODs found in vmod-vcl.md
func setupRealWorldVMODs(t *testing.T) *vmod.Registry {
	registry := vmod.NewRegistry()

	tmpDir, err := os.MkdirTemp("", "vmod_real_world_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	})

	// Crypto VMOD
	cryptoVCC := `$Module crypto 3 "Cryptographic functions"
$ABI strict

$Function STRING hex_encode(BYTES data)
$Function BYTES hash(ENUM {sha1, sha256, sha512} algorithm, STRING data)
$Function BYTES hmac(ENUM {sha1, sha256, sha512} algorithm, BYTES key, STRING data)
$Function BYTES blob(STRING data)
$Function STRING secret()
$Function INT aes_get_length()
$Function VOID aes_set_length(INT length)

$Object hmac(ENUM {sha1, sha256, sha512} algorithm, STRING key)
$Method VOID .set_key(STRING key)
$Method BYTES .digest(STRING data)`

	// S3 VMOD
	s3VCC := `$Module s3 3 "Amazon S3 authentication"
$ABI strict

$Function BOOL verify(
	STRING access_key_id,
	STRING secret_key,
	DURATION clock_skew
)`

	// YKey VMOD
	ykeyVCC := `$Module ykey 3 "YKey cache tagging"
$ABI strict

$Function INT purge(STRING keys)
$Function VOID add_key(STRING key)
$Function STRING get_hashed_keys()
$Function VOID add_hashed_keys(STRING keys)`

	// XBody VMOD
	xbodyVCC := `$Module xbody 3 "Request/response body manipulation"
$ABI strict

$Function BYTES get_req_body_hash(ENUM {sha1, sha256, sha512} algorithm)`

	// Utils VMOD
	utilsVCC := `$Module utils 3 "Utility functions"
$ABI strict

$Function STRING time_format(STRING format, BOOL local_time = 0, [TIME time])
$Function STRING newline()`

	// Probe Proxy VMOD
	probeProxyVCC := `$Module probe_proxy 3 "Probe forwarding"
$ABI strict

$Function BOOL is_probe()
$Function BACKEND self()
$Function VOID global_override(BACKEND backend)
$Function BACKEND backend()
$Function VOID force_fresh()
$Function VOID skip_health_check()
$Function DURATION timeout()`

	// Standard VCL VMODs
	stdVCC := `$Module std 3 "Standard library"
$ABI strict

$Function INT integer(STRING s, INT fallback)
$Function TIME real2time(REAL r, TIME fallback)
$Function BOOL healthy(BACKEND backend)`

	kvstoreVCC := `$Module kvstore 3 "Key-value store"
$ABI strict

$Object init(ENUM {global, task, request} scope=global)
$Method STRING .get(STRING key, STRING default="")
$Method VOID .set(STRING key, STRING value)`

	directorsVCC := `$Module directors 3 "Load balancing directors"
$ABI strict

$Object fallback()
$Method VOID .add_backend(BACKEND backend)
$Method BACKEND .backend()`

	udoVCC := `$Module udo 3 "Unified director object"
$ABI strict

$Object director()
$Method VOID .set_type(ENUM {fallback, round_robin, hash} type)
$Method BACKEND .backend()`

	vhaVCC := `$Module vha 3 "Varnish High Availability"
$ABI strict

$Function VOID log(STRING message)`

	strVCC := `$Module str 3 "String manipulation"
$ABI strict

$Function BOOL contains(STRING haystack, STRING needle)`

	headerPlusVCC := `$Module headerplus 3 "Advanced header manipulation"
$ABI strict

$Function VOID init(BEREQ bereq)
$Function VOID keep(STRING header)
$Function VOID keep_regex(STRING pattern)
$Function STRING as_list(ENUM {NAME, VALUE, BOTH} type, STRING separator, STRING join="", ENUM {LOWER, UPPER, KEEP} name_case=KEEP)
$Function VOID reset()`

	urlPlusVCC := `$Module urlplus 3 "Advanced URL manipulation"
$ABI strict

$Function VOID reset()
$Function STRING url_as_string()
$Function STRING query_as_string(BOOL query_keep_equal_sign=false)`

	gotoVCC := `$Module goto 3 "Control flow utilities"
$ABI strict`

	// Write all VCC files
	vccFiles := map[string]string{
		"crypto.vcc":      cryptoVCC,
		"s3.vcc":          s3VCC,
		"ykey.vcc":        ykeyVCC,
		"xbody.vcc":       xbodyVCC,
		"utils.vcc":       utilsVCC,
		"probe_proxy.vcc": probeProxyVCC,
		"std.vcc":         stdVCC,
		"kvstore.vcc":     kvstoreVCC,
		"directors.vcc":   directorsVCC,
		"udo.vcc":         udoVCC,
		"vha.vcc":         vhaVCC,
		"str.vcc":         strVCC,
		"headerplus.vcc":  headerPlusVCC,
		"urlplus.vcc":     urlPlusVCC,
		"goto.vcc":        gotoVCC,
	}

	for filename, content := range vccFiles {
		filePath := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", filename, err)
		}
	}

	if err := registry.LoadVCCDirectory(tmpDir); err != nil {
		t.Fatalf("Failed to load VCC files: %v", err)
	}

	return registry
}

func parseAndValidateVCL(t *testing.T, registry *vmod.Registry, vclCode string) []string {
	l := lexer.New(vclCode, "test.vcl")
	p := parser.New(l, vclCode, "test.vcl")
	program := p.ParseProgram()

	if len(p.Errors()) > 0 {
		t.Fatalf("Failed to parse VCL: %v", p.Errors()[0])
	}

	symbolTable := types.NewSymbolTable()
	validator := analyzer.NewVMODValidator(registry, symbolTable)
	return validator.Validate(program)
}

func TestCryptoVMODUsage(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "crypto_basic_usage",
			vcl: `vcl 4.0;
import crypto;
import kvstore;

sub vcl_init {
    new master_key = crypto.hmac(sha256, "secret_key");
    master_key.set_key(crypto.secret());
}`,
			expectErrors: false,
		},
		{
			name: "crypto_in_vcl_deliver",
			vcl: `vcl 4.0;
import crypto;
import std;

sub vcl_deliver {
    if (crypto.aes_get_length() > -1) {
        set resp.http.crypto-len = crypto.aes_get_length();
    }
    crypto.aes_set_length(std.integer(resp.http.X-crypto-len, 0));
}`,
			expectErrors: false,
		},
		{
			name: "crypto_hex_encoding",
			vcl: `vcl 4.0;
import crypto;

sub vcl_recv {
    set req.http.hash = crypto.hex_encode(crypto.hash(sha256, "test"));
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseAndValidateVCL(t, registry, tt.vcl)

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

func TestS3VMODUsage(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "s3_signature_verification",
			vcl: `vcl 4.0;
import s3;

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}

sub vcl_recv {
    if (req.url == "/?Action=ListUsers&Version=2010-05-08") {
        if (s3.verify(
            "AKIDEXAMPLE",
            "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
            -1s)) {
            return (pass);
        }
    }
}`,
			expectErrors: false,
		},
		{
			name: "s3_multiple_test_cases",
			vcl: `vcl 4.0;
import s3;

sub vcl_recv {
    if (req.url == "/wrong-key") {
        if (s3.verify(
            "AKIDEXAMPLE",
            "thisIsTheWrongSecretKey12345",
            -1s)) {
            return (pass);
        }
    }

    if (req.url == "/clock-skew") {
        if (s3.verify(
            "AKIDEXAMPLE",
            "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
            15m)) {
            return (pass);
        }
    }
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseAndValidateVCL(t, registry, tt.vcl)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}
		})
	}
}

func TestYKeyVMODUsage(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "ykey_basic_purging",
			vcl: `vcl 4.0;
import ykey;

sub vcl_recv {
    if (req.http.purge) {
        set req.http.npurge = ykey.purge(req.http.purge);
        return (synth(200, "Purged: " + req.http.npurge));
    }
}

sub vcl_backend_response {
    if (beresp.http.foo ~ "^bar") {
        ykey.add_key("allbar");
    }
    ykey.add_key(beresp.http.foo);
}`,
			expectErrors: false,
		},
		{
			name: "ykey_vha6_export_import",
			vcl: `vcl 4.0;
import ykey;
import vha;

sub vcl_deliver {
    if (req.method == "VHA_FETCH") {
        set resp.http.vha6-ykey = ykey.get_hashed_keys();
        if (resp.http.vha6-ykey == "") {
            unset resp.http.vha6-ykey;
        }
    }
}

sub vcl_backend_response {
    if (bereq.method == "VHA_FETCH") {
        if (beresp.http.vha6-ykey) {
            vha.log("VHA_BROADCAST PEER: YKEY import");
            ykey.add_hashed_keys(beresp.http.vha6-ykey);
        }
        unset beresp.http.vha6-ykey;
    }
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseAndValidateVCL(t, registry, tt.vcl)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}
		})
	}
}

func TestUtilsVMODUsage(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "utils_time_formatting",
			vcl: `vcl 4.0;
import utils;
import std;

sub vcl_deliver {
    set resp.http.timestamp-1 = utils.time_format("%Y%m%dT%H%M%SZ");
    set resp.http.timestamp-2 = utils.time_format("%a, %d %b %Y %H:%M:%S %Z");
    set resp.http.timestamp-3 = utils.time_format("%a, %d %b %Y %H:%M:%S %Z", time = std.real2time(-1, now));
    set resp.http.timestamp-4 = utils.time_format("%a, %d %b %Y %H:%M:%S %Z", true);
}`,
			expectErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseAndValidateVCL(t, registry, tt.vcl)

			if tt.expectErrors && len(errors) == 0 {
				t.Errorf("Expected errors but got none")
			}
			if !tt.expectErrors && len(errors) > 0 {
				t.Errorf("Expected no errors but got: %v", errors)
			}
		})
	}
}

func TestProbeProxyVMODUsage(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	vclCode := `vcl 4.0;
import kvstore;
import probe_proxy;
import std;
import str;
import udo;

sub vcl_init {
    new probe_proxy_opts = kvstore.init();
    probe_proxy.global_override(probe_proxy.self());
    new probe_proxy_gateway = udo.director();
    probe_proxy_gateway.set_type(fallback);
}

sub vcl_recv {
    if (probe_proxy.is_probe()) {
        if (str.contains(req.http.VPP-path, server.identity)) {
            return (synth(508));
        }
        set req.http.VPP-path = server.identity + " " + req.http.VPP-path;
        if (probe_proxy_opts.get("call_recv") != "true") {
            return (hash);
        }
    }
}

sub vcl_hash {
    if (probe_proxy.is_probe()) {
        hash_data(req.url);
        if (probe_proxy_opts.get("per_host", "false") == "true") {
            hash_data(req.http.Host);
        } else {
            hash_data(probe_proxy.backend());
        }
        return (lookup);
    }
}

sub vcl_backend_fetch {
    if (probe_proxy.is_probe()) {
        unset bereq.http.VPP-retry;
        if (std.healthy(probe_proxy_gateway.backend()) &&
            !bereq.http.VPP-mark) {
            set bereq.backend = probe_proxy_gateway.backend();
            set bereq.http.VPP-mark = "true";
            set bereq.http.VPP-retry = "true";
        } else {
            set bereq.backend = probe_proxy.backend();
            probe_proxy.force_fresh();
        }
        probe_proxy.skip_health_check();
        set bereq.first_byte_timeout = probe_proxy.timeout();
        set bereq.connect_timeout = probe_proxy.timeout();
        return (fetch);
    }
}`

	errors := parseAndValidateVCL(t, registry, vclCode)
	if len(errors) > 0 {
		t.Errorf("Expected no errors but got: %v", errors)
	}
}

func TestAWSSigningComplexExample(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	// This is a simplified version of the AWS signing example
	vclCode := `vcl 4.0;
import crypto;
import goto;
import headerplus;
import kvstore;
import std;
import urlplus;
import utils;
import xbody;

sub vcl_init {
    new aws_sign = kvstore.init();
}

sub aws_sign_req_body {
    if (!req.http.x-amz-content-sha256) {
        set req.http.x-amz-content-sha256 = crypto.hex_encode(xbody.get_req_body_hash(sha256));
    }
}

sub aws_sign_bereq {
    set bereq.http.x-amz-date = utils.time_format("%Y%m%dT%H%M%SZ");
    aws_sign.set("_datestamp", utils.time_format("%Y%m%d"));

    if (!bereq.http.x-amz-content-sha256) {
        set bereq.http.x-amz-content-sha256 = crypto.hex_encode(crypto.hash(sha256, ""));
    }

    headerplus.init(bereq);
    headerplus.keep("host");
    headerplus.keep_regex("^x-amz-");

    aws_sign.set("_signed_headers", headerplus.as_list(NAME, ";", name_case = LOWER));

    urlplus.reset();
    aws_sign.set("_canonical_request",
        crypto.hex_encode(crypto.hash(sha256,
            bereq.method + utils.newline() +
            urlplus.url_as_string() + utils.newline() +
            urlplus.query_as_string(query_keep_equal_sign=1) + utils.newline() +
            headerplus.as_list(BOTH, utils.newline(), ":", name_case = LOWER) + utils.newline() +
            utils.newline() +
            aws_sign.get("_signed_headers") + utils.newline() +
            bereq.http.x-amz-content-sha256)
        ));

    headerplus.reset();
}`

	errors := parseAndValidateVCL(t, registry, vclCode)
	if len(errors) > 0 {
		t.Errorf("Expected no errors but got: %v", errors)
	}
}

func TestVHA6Pattern(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	vclCode := `vcl 4.0;
import vha;
import ykey;

sub vcl_deliver {
    if (req.method == "VHA_FETCH") {
        set resp.http.vha6-data = ykey.get_hashed_keys();
        if (resp.http.vha6-data == "") {
            unset resp.http.vha6-data;
        }
    }
}

sub vcl_backend_response {
    if (bereq.method == "VHA_FETCH") {
        if (beresp.http.vha6-data) {
            vha.log("VHA_BROADCAST PEER: Data import");
            ykey.add_hashed_keys(beresp.http.vha6-data);
        }
        unset beresp.http.vha6-data;
    }
}`

	errors := parseAndValidateVCL(t, registry, vclCode)
	if len(errors) > 0 {
		t.Errorf("Expected no errors but got: %v", errors)
	}
}

func TestVMODErrorCases(t *testing.T) {
	registry := setupRealWorldVMODs(t)

	tests := []struct {
		name          string
		vcl           string
		expectErrors  bool
		errorContains []string
	}{
		{
			name: "missing_import",
			vcl: `vcl 4.0;

sub vcl_recv {
    if (crypto.aes_get_length() > -1) {
        set req.http.test = "value";
    }
}`,
			expectErrors:  true,
			errorContains: []string{"not imported"},
		},
		{
			name: "unknown_vmod",
			vcl: `vcl 4.0;
import nonexistent;

sub vcl_recv {
    set req.http.test = "value";
}`,
			expectErrors:  true,
			errorContains: []string{"not available"},
		},
		{
			name: "object_without_import",
			vcl: `vcl 4.0;

sub vcl_init {
    new master_key = crypto.hmac(sha256, "secret");
}`,
			expectErrors:  true,
			errorContains: []string{"not imported"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := parseAndValidateVCL(t, registry, tt.vcl)

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
