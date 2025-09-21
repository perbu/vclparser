package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/vmod"
)

// setupTestRegistry creates a test VMOD registry with common VCC definitions
func setupTestRegistry(t *testing.T) *vmod.Registry {
	registry := vmod.NewRegistry()

	// Create temporary directory for VCC files
	tmpDir, err := os.MkdirTemp("", "vcc_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("Failed to remove temp directory: %v", err)
		}
	})

	// Create std.vcc with comprehensive functions
	stdVCC := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING_LIST s)
$Function VOID log(STRING_LIST s)
$Function REAL random(REAL lo, REAL hi)
$Function BOOL file_exists(STRING path)
$Function TIME real2time(REAL r, TIME base)
$Function REAL time2real(TIME t)
$Function STRING time2integer(TIME t)
$Function STRING real2integer(REAL r)
$Function REAL integer2real(STRING s)
$Function TIME integer2time(STRING s)
$Function BOOL healthy(BACKEND be)
$Function INT port(IP i)
$Function VOID rollback(HTTP h)`

	stdFile := filepath.Join(tmpDir, "std.vcc")
	err = os.WriteFile(stdFile, []byte(stdVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write std.vcc: %v", err)
	}

	// Create directors.vcc
	directorsVCC := `$Module directors 3 "Directors module"
$ABI strict

$Object round_robin()
$Method BACKEND .backend()
$Method VOID .add_backend(BACKEND)

$Object hash()
$Method BACKEND .backend([STRING key])
$Method VOID .add_backend(BACKEND, REAL weight = 1.0)`

	directorsFile := filepath.Join(tmpDir, "directors.vcc")
	err = os.WriteFile(directorsFile, []byte(directorsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write directors.vcc: %v", err)
	}

	// Create utils.vcc for regression tests
	utilsVCC := `$Module utils 3 "Utility functions for VCL"
$ABI strict
$Function STRING time_format(STRING format, BOOL local_time = 0, [TIME time])
Format the time according to format.`

	utilsFile := filepath.Join(tmpDir, "utils.vcc")
	err = os.WriteFile(utilsFile, []byte(utilsVCC), 0644)
	if err != nil {
		t.Fatalf("Failed to write utils.vcc: %v", err)
	}

	// Load VCC files into registry
	err = registry.LoadVCCFile(stdFile)
	if err != nil {
		t.Fatalf("Failed to load std.vcc: %v", err)
	}
	err = registry.LoadVCCFile(directorsFile)
	if err != nil {
		t.Fatalf("Failed to load directors.vcc: %v", err)
	}
	err = registry.LoadVCCFile(utilsFile)
	if err != nil {
		t.Fatalf("Failed to load utils.vcc: %v", err)
	}

	return registry
}
