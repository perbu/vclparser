package so

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/vcc"
)

func TestLoadModuleFromSO_ParsesStdFunctions(t *testing.T) {
	stdPath := mustFindFixture(t,
		filepath.Join("..", "..", "vmods", "libvmod_std.so"),
		filepath.Join("vmods", "libvmod_std.so"),
	)

	module, err := LoadModuleFromSO(stdPath)
	if err != nil {
		t.Fatalf("LoadModuleFromSO failed: %v", err)
	}

	if module.Name != "std" {
		t.Fatalf("expected module name std, got %q", module.Name)
	}

	toupper := module.FindFunction("toupper")
	if toupper == nil {
		t.Fatalf("expected function toupper to be present")
	}
	if toupper.ReturnType != vcc.TypeString {
		t.Fatalf("expected toupper return type STRING, got %s", toupper.ReturnType)
	}
	if len(toupper.Parameters) != 1 {
		t.Fatalf("expected toupper to have 1 parameter, got %d", len(toupper.Parameters))
	}
	if toupper.Parameters[0].Type != vcc.TypeStringList {
		t.Fatalf("expected toupper parameter type STRING_LIST, got %s", toupper.Parameters[0].Type)
	}

	setIPTOS := module.FindFunction("set_ip_tos")
	if setIPTOS == nil {
		t.Fatalf("expected function set_ip_tos to be present")
	}
	if !contains(setIPTOS.Restrictions, "client") {
		t.Fatalf("expected set_ip_tos to have client restriction, got %v", setIPTOS.Restrictions)
	}
}

func TestLoadModuleFromSO_ParsesObjectsAndMethods(t *testing.T) {
	directorsPath := mustFindFixture(t,
		filepath.Join("..", "..", "vmods", "libvmod_directors.so"),
		filepath.Join("vmods", "libvmod_directors.so"),
	)

	module, err := LoadModuleFromSO(directorsPath)
	if err != nil {
		t.Fatalf("LoadModuleFromSO failed: %v", err)
	}

	roundRobin := module.FindObject("round_robin")
	if roundRobin == nil {
		t.Fatalf("expected object round_robin to be present")
	}
	if len(roundRobin.Constructor) != 0 {
		t.Fatalf("expected round_robin constructor to have 0 parameters, got %d", len(roundRobin.Constructor))
	}

	addBackend := roundRobin.FindMethod("add_backend")
	if addBackend == nil {
		t.Fatalf("expected round_robin.add_backend method to be present")
	}
	if addBackend.ReturnType != vcc.TypeVoid {
		t.Fatalf("expected add_backend return type VOID, got %s", addBackend.ReturnType)
	}
	if len(addBackend.Parameters) != 1 {
		t.Fatalf("expected add_backend to have 1 parameter, got %d", len(addBackend.Parameters))
	}
	if addBackend.Parameters[0].Type != vcc.TypeBackend {
		t.Fatalf("expected add_backend parameter type BACKEND, got %s", addBackend.Parameters[0].Type)
	}
}

func TestLoadModuleFromSO_ParsesEnumAndOptionalParameters(t *testing.T) {
	udoPath := mustFindFixture(t,
		filepath.Join("..", "..", "vmods", "libvmod_udo.so"),
		filepath.Join("vmods", "libvmod_udo.so"),
	)

	module, err := LoadModuleFromSO(udoPath)
	if err != nil {
		t.Fatalf("LoadModuleFromSO failed: %v", err)
	}

	director := module.FindObject("director")
	if director == nil {
		t.Fatalf("expected object director to be present")
	}

	setType := director.FindMethod("set_type")
	if setType == nil {
		t.Fatalf("expected method set_type to be present")
	}
	if len(setType.Parameters) != 1 {
		t.Fatalf("expected set_type to have 1 parameter, got %d", len(setType.Parameters))
	}
	if setType.Parameters[0].Type != vcc.TypeEnum {
		t.Fatalf("expected set_type parameter type ENUM, got %s", setType.Parameters[0].Type)
	}
	if setType.Parameters[0].Enum == nil || len(setType.Parameters[0].Enum.Values) == 0 {
		t.Fatalf("expected set_type enum values to be present")
	}

	reset := director.FindMethod("reset")
	if reset == nil {
		t.Fatalf("expected method reset to be present")
	}
	if len(reset.Parameters) < 2 {
		t.Fatalf("expected reset to have at least 2 parameters, got %d", len(reset.Parameters))
	}
	if !reset.Parameters[1].Optional {
		t.Fatalf("expected reset second parameter to be optional")
	}
}

func TestLoadModuleFromSO_RejectsNonELFBinary(t *testing.T) {
	machoPath := mustFindFixture(t,
		filepath.Join("testdata", "macho", "libvmod_std.so"),
		filepath.Join("pkg", "so", "testdata", "macho", "libvmod_std.so"),
	)

	_, err := LoadModuleFromSO(machoPath)
	if err == nil {
		t.Fatalf("expected error when loading non-ELF shared object")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "elf") {
		t.Fatalf("expected ELF-related error, got: %v", err)
	}
}

func mustFindFixture(t *testing.T, candidates ...string) string {
	t.Helper()

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	t.Skipf("fixture not found in any candidate path: %v", candidates)
	return ""
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
