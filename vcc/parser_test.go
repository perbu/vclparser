package vcc

import (
	"strings"
	"testing"
)

func TestParseSimpleModule(t *testing.T) {
	vccContent := `$Module std 3 "Standard library"
$ABI strict

$Function STRING toupper(STRING_LIST s)

Description
Converts the string to uppercase.

$Function VOID log(STRING_LIST s)

Logs the string to VSL.`

	parser := NewParser(strings.NewReader(vccContent))
	module, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if module.Name != "std" {
		t.Errorf("Expected module name 'std', got '%s'", module.Name)
	}

	if module.Version != 3 {
		t.Errorf("Expected version 3, got %d", module.Version)
	}

	if module.ABI != "strict" {
		t.Errorf("Expected ABI 'strict', got '%s'", module.ABI)
	}

	if len(module.Functions) != 2 {
		t.Errorf("Expected 2 functions, got %d", len(module.Functions))
	}

	// Test first function
	toupper := module.Functions[0]
	if toupper.Name != "toupper" {
		t.Errorf("Expected function name 'toupper', got '%s'", toupper.Name)
	}

	if toupper.ReturnType != TypeString {
		t.Errorf("Expected return type STRING, got %s", toupper.ReturnType)
	}

	if len(toupper.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(toupper.Parameters))
	}

	param := toupper.Parameters[0]
	if param.Name != "s" {
		t.Errorf("Expected parameter name 's', got '%s'", param.Name)
	}

	if param.Type != TypeStringList {
		t.Errorf("Expected parameter type STRING_LIST, got %s", param.Type)
	}

	// Test second function
	log := module.Functions[1]
	if log.Name != "log" {
		t.Errorf("Expected function name 'log', got '%s'", log.Name)
	}

	if log.ReturnType != TypeVoid {
		t.Errorf("Expected return type VOID, got %s", log.ReturnType)
	}
}

func TestParseObjectWithMethods(t *testing.T) {
	vccContent := `$Module directors 3 "Directors module"

$Object round_robin()

Description
	Create a round robin director.

$Method VOID .add_backend(BACKEND)

Description
	Add a backend to the round-robin director.

$Method BACKEND .backend()

Description
	Get a backend from the round-robin director.`

	parser := NewParser(strings.NewReader(vccContent))
	module, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(module.Objects) != 1 {
		t.Errorf("Expected 1 object, got %d", len(module.Objects))
	}

	obj := module.Objects[0]
	if obj.Name != "round_robin" {
		t.Errorf("Expected object name 'round_robin', got '%s'", obj.Name)
	}

	if len(obj.Constructor) != 0 {
		t.Errorf("Expected 0 constructor parameters, got %d", len(obj.Constructor))
	}

	if len(obj.Methods) != 2 {
		t.Errorf("Expected 2 methods, got %d", len(obj.Methods))
	}

	// Test add_backend method
	addBackend := obj.Methods[0]
	if addBackend.Name != "add_backend" {
		t.Errorf("Expected method name 'add_backend', got '%s'", addBackend.Name)
	}

	if addBackend.ReturnType != TypeVoid {
		t.Errorf("Expected return type VOID, got %s", addBackend.ReturnType)
	}

	if len(addBackend.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(addBackend.Parameters))
	}

	if addBackend.Parameters[0].Type != TypeBackend {
		t.Errorf("Expected parameter type BACKEND, got %s", addBackend.Parameters[0].Type)
	}

	// Test backend method
	backend := obj.Methods[1]
	if backend.Name != "backend" {
		t.Errorf("Expected method name 'backend', got '%s'", backend.Name)
	}

	if backend.ReturnType != TypeBackend {
		t.Errorf("Expected return type BACKEND, got %s", backend.ReturnType)
	}
}

func TestParseEnumParameters(t *testing.T) {
	vccContent := `$Module blob 3 "Blob module"

$Function STRING encode(ENUM {IDENTITY, BASE64, HEX} encoding="IDENTITY", BLOB blob)

Encode a blob using the specified encoding.`

	parser := NewParser(strings.NewReader(vccContent))
	module, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(module.Functions) != 1 {
		t.Errorf("Expected 1 function, got %d", len(module.Functions))
	}

	function := module.Functions[0]
	if len(function.Parameters) != 2 {
		t.Errorf("Expected 2 parameters, got %d", len(function.Parameters))
	}

	// Test enum parameter
	enumParam := function.Parameters[0]
	if enumParam.Type != TypeEnum {
		t.Errorf("Expected parameter type ENUM, got %s", enumParam.Type)
	}

	if enumParam.Enum == nil {
		t.Fatal("Expected enum definition, got nil")
	}

	expectedValues := []string{"IDENTITY", "BASE64", "HEX"}
	if len(enumParam.Enum.Values) != len(expectedValues) {
		t.Errorf("Expected %d enum values, got %d", len(expectedValues), len(enumParam.Enum.Values))
	}

	for i, expected := range expectedValues {
		if i < len(enumParam.Enum.Values) && enumParam.Enum.Values[i] != expected {
			t.Errorf("Expected enum value '%s' at index %d, got '%s'", expected, i, enumParam.Enum.Values[i])
		}
	}

	if enumParam.DefaultValue != "IDENTITY" {
		t.Errorf("Expected default value 'IDENTITY', got '%s'", enumParam.DefaultValue)
	}

	// Test blob parameter
	blobParam := function.Parameters[1]
	if blobParam.Type != TypeBlob {
		t.Errorf("Expected parameter type BLOB, got %s", blobParam.Type)
	}

	if blobParam.Name != "blob" {
		t.Errorf("Expected parameter name 'blob', got '%s'", blobParam.Name)
	}
}

func TestFunctionValidation(t *testing.T) {
	function := Function{
		Name:       "test_function",
		ReturnType: TypeString,
		Parameters: []Parameter{
			{Name: "str", Type: TypeString},
			{Name: "num", Type: TypeInt},
			{Name: "optional", Type: TypeString, Optional: true},
		},
	}

	// Test valid call
	err := function.ValidateCall([]VCCType{TypeString, TypeInt})
	if err != nil {
		t.Errorf("Valid call should not produce error: %v", err)
	}

	// Test call with optional parameter
	err = function.ValidateCall([]VCCType{TypeString, TypeInt, TypeString})
	if err != nil {
		t.Errorf("Valid call with optional parameter should not produce error: %v", err)
	}

	// Test insufficient arguments
	err = function.ValidateCall([]VCCType{TypeString})
	if err == nil {
		t.Error("Call with insufficient arguments should produce error")
	}

	// Test too many arguments
	err = function.ValidateCall([]VCCType{TypeString, TypeInt, TypeString, TypeInt})
	if err == nil {
		t.Error("Call with too many arguments should produce error")
	}

	// Test wrong argument type
	err = function.ValidateCall([]VCCType{TypeInt, TypeString})
	if err == nil {
		t.Error("Call with wrong argument types should produce error")
	}
}

func TestModuleFindFunctions(t *testing.T) {
	module := Module{
		Name: "test",
		Functions: []Function{
			{Name: "func1", ReturnType: TypeString},
			{Name: "func2", ReturnType: TypeInt},
		},
		Objects: []Object{
			{Name: "obj1"},
			{Name: "obj2"},
		},
	}

	// Test finding existing function
	function := module.FindFunction("func1")
	if function == nil {
		t.Error("Should find existing function")
	}
	if function.Name != "func1" {
		t.Errorf("Expected function name 'func1', got '%s'", function.Name)
	}

	// Test finding non-existing function
	function = module.FindFunction("func3")
	if function != nil {
		t.Error("Should not find non-existing function")
	}

	// Test finding existing object
	object := module.FindObject("obj1")
	if object == nil {
		t.Error("Should find existing object")
	}
	if object.Name != "obj1" {
		t.Errorf("Expected object name 'obj1', got '%s'", object.Name)
	}

	// Test finding non-existing object
	object = module.FindObject("obj3")
	if object != nil {
		t.Error("Should not find non-existing object")
	}
}

func TestParseVCCType(t *testing.T) {
	tests := []struct {
		input    string
		expected VCCType
		hasEnum  bool
	}{
		{"STRING", TypeString, false},
		{"INT", TypeInt, false},
		{"REAL", TypeReal, false},
		{"BOOL", TypeBool, false},
		{"STRING_LIST", TypeStringList, false},
		{"ENUM {A, B, C}", TypeEnum, true},
	}

	for _, test := range tests {
		vccType, enum, err := ParseVCCType(test.input)
		if err != nil {
			t.Errorf("ParseVCCType(%s) failed: %v", test.input, err)
			continue
		}

		if vccType != test.expected {
			t.Errorf("ParseVCCType(%s): expected %s, got %s", test.input, test.expected, vccType)
		}

		if test.hasEnum && enum == nil {
			t.Errorf("ParseVCCType(%s): expected enum definition, got nil", test.input)
		}

		if !test.hasEnum && enum != nil {
			t.Errorf("ParseVCCType(%s): expected no enum definition, got %v", test.input, enum)
		}
	}
}

func TestDurationDefaultValues(t *testing.T) {
	vccContent := `$Module test 3 "Test module for duration defaults"
$ABI strict

$Function BOOL verify_timeout(STRING key, DURATION timeout = -1s)
$Function VOID set_cache_ttl(STRING key, DURATION ttl = 300s)
$Function INT get_delay(DURATION delay = 30m)
$Function REAL calculate(DURATION window = 2h)`

	parser := NewParser(strings.NewReader(vccContent))
	module, err := parser.Parse()

	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(module.Functions) != 4 {
		t.Fatalf("Expected 4 functions, got %d", len(module.Functions))
	}

	// Test negative duration default
	verifyTimeout := module.Functions[0]
	if len(verifyTimeout.Parameters) != 2 {
		t.Fatalf("Expected 2 parameters in verify_timeout, got %d", len(verifyTimeout.Parameters))
	}
	timeoutParam := verifyTimeout.Parameters[1]
	if timeoutParam.Type != TypeDuration {
		t.Errorf("Expected DURATION type, got %s", timeoutParam.Type)
	}
	if timeoutParam.DefaultValue != "-1s" {
		t.Errorf("Expected default value '-1s', got '%s'", timeoutParam.DefaultValue)
	}

	// Test positive duration default
	setCacheTtl := module.Functions[1]
	ttlParam := setCacheTtl.Parameters[1]
	if ttlParam.DefaultValue != "300s" {
		t.Errorf("Expected default value '300s', got '%s'", ttlParam.DefaultValue)
	}

	// Test minute duration default
	getDelay := module.Functions[2]
	delayParam := getDelay.Parameters[0]
	if delayParam.DefaultValue != "30m" {
		t.Errorf("Expected default value '30m', got '%s'", delayParam.DefaultValue)
	}

	// Test hour duration default
	calculate := module.Functions[3]
	windowParam := calculate.Parameters[0]
	if windowParam.DefaultValue != "2h" {
		t.Errorf("Expected default value '2h', got '%s'", windowParam.DefaultValue)
	}
}
