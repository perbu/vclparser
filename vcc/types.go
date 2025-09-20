package vcc

import (
	"fmt"
	"strings"
)

// VCCType represents a VCC data type
type VCCType string

const (
	TypeString     VCCType = "STRING"
	TypeInt        VCCType = "INT"
	TypeReal       VCCType = "REAL"
	TypeBool       VCCType = "BOOL"
	TypeBlob       VCCType = "BLOB"
	TypeBackend    VCCType = "BACKEND"
	TypeHeader     VCCType = "HEADER"
	TypeStringList VCCType = "STRING_LIST"
	TypeStrands    VCCType = "STRANDS"
	TypeDuration   VCCType = "DURATION"
	TypeBytes      VCCType = "BYTES"
	TypeIP         VCCType = "IP"
	TypeTime       VCCType = "TIME"
	TypeVoid       VCCType = "VOID"
	TypePrivCall   VCCType = "PRIV_CALL"
	TypePrivVCL    VCCType = "PRIV_VCL"
	TypePrivTask   VCCType = "PRIV_TASK"
	TypeACL        VCCType = "ACL"
	TypeProbe      VCCType = "PROBE"
	TypeSubroutine VCCType = "SUB"
	TypeEnum       VCCType = "ENUM"
	TypeHTTP       VCCType = "HTTP"
	TypeStevedore  VCCType = "STEVEDORE"
	TypePrivTop    VCCType = "PRIV_TOP"
	TypeBereq      VCCType = "BEREQ"
)

// IsCompatibleType checks if two VCC types are compatible
func IsCompatibleType(actual, expected VCCType) bool {
	if actual == expected {
		return true
	}

	// Allow INT to REAL coercion (common in VCL)
	if expected == TypeReal && actual == TypeInt {
		return true
	}

	// Allow INT to BOOL coercion (common in C-style languages: 1=true, 0=false)
	if expected == TypeBool && actual == TypeInt {
		return true
	}

	// HTTP objects are compatible with their specific types
	if actual == TypeHTTP && (expected == TypeBereq || expected == "REQ" || expected == "RESP" || expected == "BERESP") {
		return true
	}

	// STRING_LIST can accept STRING
	if expected == TypeStringList && actual == TypeString {
		return true
	}

	// STRANDS can accept STRING or STRING_LIST
	if expected == TypeStrands && (actual == TypeString || actual == TypeStringList) {
		return true
	}

	return false
}

// Enum represents an enum definition in VCC
type Enum struct {
	Values       []string
	DefaultValue string
}

// Parameter represents a function/method parameter
type Parameter struct {
	Name         string
	Type         VCCType
	Enum         *Enum  // Non-nil for ENUM types
	DefaultValue string // Optional default value
	Optional     bool   // Whether parameter is optional
}

// Function represents a VCC function definition
type Function struct {
	Name         string
	ReturnType   VCCType
	Parameters   []Parameter
	Description  string
	Examples     []string
	Restrictions []string // VCL contexts where function can be used
}

// Method represents a VCC object method
type Method struct {
	Name         string
	ReturnType   VCCType
	Parameters   []Parameter
	Description  string
	Examples     []string
	Restrictions []string
}

// Object represents a VCC object definition
type Object struct {
	Name        string
	Constructor []Parameter // Parameters for object instantiation
	Methods     []Method
	Description string
	Examples    []string
}

// Event represents a VCC event handler
type Event struct {
	Name        string
	Description string
}

// Module represents a complete VCC module definition
type Module struct {
	Name        string
	Version     int
	Description string
	Functions   []Function
	Objects     []Object
	Events      []Event
	ABI         string // ABI specification
}

// String returns a string representation of the module
func (m *Module) String() string {
	return fmt.Sprintf("Module %s v%d (%d functions, %d objects)",
		m.Name, m.Version, len(m.Functions), len(m.Objects))
}

// FindFunction finds a function by name
func (m *Module) FindFunction(name string) *Function {
	//nolint:nilaway // receiver m is validated by caller
	for i := range m.Functions {
		if m.Functions[i].Name == name {
			return &m.Functions[i]
		}
	}
	return nil
}

// FindObject finds an object by name
func (m *Module) FindObject(name string) *Object {
	//nolint:nilaway // receiver m is validated by caller
	for i := range m.Objects {
		if m.Objects[i].Name == name {
			return &m.Objects[i]
		}
	}
	return nil
}

// FindMethod finds a method on an object by name
func (o *Object) FindMethod(name string) *Method {
	for i := range o.Methods {
		if o.Methods[i].Name == name {
			return &o.Methods[i]
		}
	}
	return nil
}

// ValidateCall validates a function call against the function signature
func (f *Function) ValidateCall(args []VCCType) error {
	// Check if we have the required number of arguments
	requiredParams := 0
	for _, param := range f.Parameters {
		if !param.Optional && param.DefaultValue == "" {
			requiredParams++
		}
	}

	if len(args) < requiredParams {
		return fmt.Errorf("function %s requires at least %d arguments, got %d",
			f.Name, requiredParams, len(args))
	}

	if len(args) > len(f.Parameters) {
		return fmt.Errorf("function %s accepts at most %d arguments, got %d",
			f.Name, len(f.Parameters), len(args))
	}

	// Validate argument types
	for i, arg := range args {
		expected := f.Parameters[i].Type
		if !f.isCompatibleType(arg, expected) {
			return fmt.Errorf("function %s argument %d: expected %s, got %s",
				f.Name, i+1, expected, arg)
		}
	}

	return nil
}

// ValidateCall validates a method call against the method signature
func (m *Method) ValidateCall(args []VCCType) error {
	// Similar validation logic as Function.ValidateCall
	requiredParams := 0
	for _, param := range m.Parameters {
		if !param.Optional && param.DefaultValue == "" {
			requiredParams++
		}
	}

	if len(args) < requiredParams {
		return fmt.Errorf("method %s requires at least %d arguments, got %d",
			m.Name, requiredParams, len(args))
	}

	if len(args) > len(m.Parameters) {
		return fmt.Errorf("method %s accepts at most %d arguments, got %d",
			m.Name, len(m.Parameters), len(args))
	}

	// Validate argument types
	for i, arg := range args {
		expected := m.Parameters[i].Type
		if !m.isCompatibleType(arg, expected) {
			return fmt.Errorf("method %s argument %d: expected %s, got %s",
				m.Name, i+1, expected, arg)
		}
	}

	return nil
}

// ValidateConstruction validates object construction against constructor parameters
func (o *Object) ValidateConstruction(args []VCCType) error {
	// Check if we have the required number of arguments
	requiredParams := 0
	for _, param := range o.Constructor {
		if !param.Optional && param.DefaultValue == "" {
			requiredParams++
		}
	}

	if len(args) < requiredParams {
		return fmt.Errorf("object %s constructor requires at least %d arguments, got %d",
			o.Name, requiredParams, len(args))
	}

	if len(args) > len(o.Constructor) {
		return fmt.Errorf("object %s constructor accepts at most %d arguments, got %d",
			o.Name, len(o.Constructor), len(args))
	}

	// Validate argument types
	for i, arg := range args {
		expected := o.Constructor[i].Type
		if !o.isCompatibleType(arg, expected) {
			return fmt.Errorf("object %s constructor argument %d: expected %s, got %s",
				o.Name, i+1, expected, arg)
		}
	}

	return nil
}

// isCompatibleType checks if two types are compatible
func (f *Function) isCompatibleType(actual, expected VCCType) bool {
	return IsCompatibleType(actual, expected)
}

// isCompatibleType checks if two types are compatible for methods
func (m *Method) isCompatibleType(actual, expected VCCType) bool {
	return IsCompatibleType(actual, expected)
}

// isCompatibleType checks if two types are compatible for object constructors
func (o *Object) isCompatibleType(actual, expected VCCType) bool {
	return IsCompatibleType(actual, expected)
}

// ParseVCCType parses a VCC type string, handling complex types like ENUM
func ParseVCCType(typeStr string) (VCCType, *Enum, error) {
	typeStr = strings.TrimSpace(typeStr)

	// Handle ENUM types
	if strings.HasPrefix(typeStr, "ENUM") {
		return TypeEnum, parseEnum(typeStr), nil
	}

	// Handle standard types
	switch strings.ToUpper(typeStr) {
	case "STRING":
		return TypeString, nil, nil
	case "STRING_LIST":
		return TypeStringList, nil, nil
	case "INT":
		return TypeInt, nil, nil
	case "REAL":
		return TypeReal, nil, nil
	case "BOOL":
		return TypeBool, nil, nil
	case "BLOB":
		return TypeBlob, nil, nil
	case "BACKEND":
		return TypeBackend, nil, nil
	case "HEADER":
		return TypeHeader, nil, nil
	case "STRANDS":
		return TypeStrands, nil, nil
	case "DURATION":
		return TypeDuration, nil, nil
	case "BYTES":
		return TypeBytes, nil, nil
	case "IP":
		return TypeIP, nil, nil
	case "TIME":
		return TypeTime, nil, nil
	case "VOID":
		return TypeVoid, nil, nil
	case "PRIV_CALL":
		return TypePrivCall, nil, nil
	case "PRIV_VCL":
		return TypePrivVCL, nil, nil
	case "PRIV_TASK":
		return TypePrivTask, nil, nil
	case "ACL":
		return TypeACL, nil, nil
	case "PROBE":
		return TypeProbe, nil, nil
	case "SUB":
		return TypeSubroutine, nil, nil
	case "HTTP":
		return TypeHTTP, nil, nil
	case "STEVEDORE":
		return TypeStevedore, nil, nil
	case "PRIV_TOP":
		return TypePrivTop, nil, nil
	case "BEREQ":
		return TypeBereq, nil, nil
	default:
		return VCCType(typeStr), nil, fmt.Errorf("unknown VCC type: %s", typeStr)
	}
}

// parseEnum parses an ENUM type definition
func parseEnum(enumStr string) *Enum {
	// Extract values from "ENUM {VAL1, VAL2, VAL3}"
	start := strings.Index(enumStr, "{")
	end := strings.LastIndex(enumStr, "}")

	if start == -1 || end == -1 || start >= end {
		return &Enum{Values: []string{}}
	}

	valuesStr := enumStr[start+1 : end]
	values := []string{}

	for _, val := range strings.Split(valuesStr, ",") {
		val = strings.TrimSpace(val)
		if val != "" {
			values = append(values, val)
		}
	}

	return &Enum{Values: values}
}
