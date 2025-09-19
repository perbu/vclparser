package types

import "fmt"

// Type represents a VCL type
type Type interface {
	String() string
	IsAssignableTo(other Type) bool
	IsComparable() bool
}

// BasicType represents basic VCL types
type BasicType struct {
	Name string
}

func (t *BasicType) String() string {
	return t.Name
}

func (t *BasicType) IsAssignableTo(other Type) bool {
	if ot, ok := other.(*BasicType); ok {
		return t.Name == ot.Name
	}
	return false
}

func (t *BasicType) IsComparable() bool {
	return true
}

// Built-in VCL types
var (
	String   = &BasicType{Name: "STRING"}
	Int      = &BasicType{Name: "INT"}
	Real     = &BasicType{Name: "REAL"}
	Bool     = &BasicType{Name: "BOOL"}
	Time     = &BasicType{Name: "TIME"}
	Duration = &BasicType{Name: "DURATION"}
	IP       = &BasicType{Name: "IP"}
	Backend  = &BasicType{Name: "BACKEND"}
	ACL      = &BasicType{Name: "ACL"}
	Probe    = &BasicType{Name: "PROBE"}
	Header   = &BasicType{Name: "HEADER"}
	Void     = &BasicType{Name: "VOID"}
	Module   = &BasicType{Name: "MODULE"}
	Object   = &BasicType{Name: "OBJECT"}
	Bytes    = &BasicType{Name: "BYTES"}
)

// HeaderType represents header variables
type HeaderType struct {
	HeaderName string
}

func (t *HeaderType) String() string {
	return fmt.Sprintf("HEADER(%s)", t.HeaderName)
}

func (t *HeaderType) IsAssignableTo(other Type) bool {
	if ot, ok := other.(*HeaderType); ok {
		return t.HeaderName == ot.HeaderName
	}
	if ot, ok := other.(*BasicType); ok {
		return ot.Name == "HEADER" || ot.Name == "STRING"
	}
	return false
}

func (t *HeaderType) IsComparable() bool {
	return true
}

// FunctionType represents function types
type FunctionType struct {
	Parameters []Type
	ReturnType Type
}

func (t *FunctionType) String() string {
	params := make([]string, len(t.Parameters))
	for i, p := range t.Parameters {
		params[i] = p.String()
	}
	return fmt.Sprintf("(%s) -> %s", params, t.ReturnType.String())
}

func (t *FunctionType) IsAssignableTo(other Type) bool {
	if ot, ok := other.(*FunctionType); ok {
		if len(t.Parameters) != len(ot.Parameters) {
			return false
		}
		for i, param := range t.Parameters {
			if !param.IsAssignableTo(ot.Parameters[i]) {
				return false
			}
		}
		return t.ReturnType.IsAssignableTo(ot.ReturnType)
	}
	return false
}

func (t *FunctionType) IsComparable() bool {
	return false
}

// ArrayType represents array types
type ArrayType struct {
	ElementType Type
}

func (t *ArrayType) String() string {
	return fmt.Sprintf("[]%s", t.ElementType.String())
}

func (t *ArrayType) IsAssignableTo(other Type) bool {
	if ot, ok := other.(*ArrayType); ok {
		return t.ElementType.IsAssignableTo(ot.ElementType)
	}
	return false
}

func (t *ArrayType) IsComparable() bool {
	return false
}

// TypeFromString returns a Type from a string representation
func TypeFromString(name string) Type {
	switch name {
	case "STRING":
		return String
	case "INT":
		return Int
	case "REAL":
		return Real
	case "BOOL":
		return Bool
	case "TIME":
		return Time
	case "DURATION":
		return Duration
	case "IP":
		return IP
	case "BACKEND":
		return Backend
	case "ACL":
		return ACL
	case "PROBE":
		return Probe
	case "HEADER":
		return Header
	case "VOID":
		return Void
	default:
		return &BasicType{Name: name}
	}
}

// IsNumeric returns true if the type is numeric
func IsNumeric(t Type) bool {
	if bt, ok := t.(*BasicType); ok {
		return bt.Name == "INT" || bt.Name == "REAL"
	}
	return false
}

// CanCast returns true if from type can be cast to to type
func CanCast(from, to Type) bool {
	if from.IsAssignableTo(to) {
		return true
	}

	// Special casting rules
	if IsNumeric(from) && IsNumeric(to) {
		return true
	}

	// String conversions
	if to.String() == "STRING" {
		return true // Most types can be converted to string
	}

	return false
}
