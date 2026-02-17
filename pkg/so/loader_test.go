package so

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/perbu/vclparser/pkg/vcc"
)

func TestLoadModuleFromSO_ParsesStdFunctions(t *testing.T) {
	stdPath := mustFindFixture(t,
		filepath.Join("testdata", "elf", "libvmod_std.so"),
		filepath.Join("pkg", "so", "testdata", "elf", "libvmod_std.so"),
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
	if toupper.Parameters[0].Type != vcc.TypeStringList && toupper.Parameters[0].Type != vcc.TypeStrands {
		t.Fatalf("expected toupper parameter type STRING_LIST or STRANDS, got %s", toupper.Parameters[0].Type)
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
		filepath.Join("testdata", "elf", "libvmod_directors.so"),
		filepath.Join("pkg", "so", "testdata", "elf", "libvmod_directors.so"),
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
	directorsPath := mustFindFixture(t,
		filepath.Join("testdata", "elf", "libvmod_directors.so"),
		filepath.Join("pkg", "so", "testdata", "elf", "libvmod_directors.so"),
	)

	module, err := LoadModuleFromSO(directorsPath)
	if err != nil {
		t.Fatalf("LoadModuleFromSO failed: %v", err)
	}

	shard := module.FindObject("shard")
	if shard == nil {
		t.Fatalf("expected object shard to be present")
	}

	backend := shard.FindMethod("backend")
	if backend == nil {
		t.Fatalf("expected method backend to be present")
	}
	if len(backend.Parameters) < 2 {
		t.Fatalf("expected backend to have at least 2 parameters, got %d", len(backend.Parameters))
	}
	if backend.Parameters[0].Type != vcc.TypeEnum {
		t.Fatalf("expected backend first parameter type ENUM, got %s", backend.Parameters[0].Type)
	}
	if backend.Parameters[0].Enum == nil || len(backend.Parameters[0].Enum.Values) == 0 {
		t.Fatalf("expected backend first enum values to be present")
	}
	if !backend.Parameters[1].Optional {
		t.Fatalf("expected backend second parameter to be optional")
	}
}

func TestLoadModuleFromSO_ParsesMachOBinary(t *testing.T) {
	machoPath := mustFindFixture(t,
		filepath.Join("testdata", "macho", "libvmod_std.so"),
		filepath.Join("pkg", "so", "testdata", "macho", "libvmod_std.so"),
	)

	module, err := LoadModuleFromSO(machoPath)
	if err != nil {
		t.Fatalf("expected Mach-O shared object to load, got: %v", err)
	}
	if module.Name != "std" {
		t.Fatalf("expected module name std, got %q", module.Name)
	}
	if module.FindFunction("toupper") == nil {
		t.Fatalf("expected function toupper to be present in Mach-O fixture")
	}
}

func TestLoadModuleFromSO_RejectsUnsupportedBinary(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "not-a-shared-object.so")
	if err := os.WriteFile(filePath, []byte("not-a-binary"), 0o644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := LoadModuleFromSO(filePath)
	if err == nil {
		t.Fatalf("expected error when loading unsupported shared object format")
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
