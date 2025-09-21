package metadata

import "strings"

// VCLMetadata represents the complete VCL metadata from varnishd
type VCLMetadata struct {
	VCLMethods       map[string]VCLMethod   `json:"vcl_methods"`
	VCLVariables     map[string]VCLVariable `json:"vcl_variables"`
	VCLTypes         map[string]VCLType     `json:"vcl_types"`
	VCLTokens        map[string]string      `json:"vcl_tokens"`
	StorageVariables []StorageVariable      `json:"storage_variables"`
}

// VCLMethod represents a VCL method with its context and allowed returns
type VCLMethod struct {
	Context        string   `json:"context"`         // "C" (client), "B" (backend), "H" (housekeeping)
	AllowedReturns []string `json:"allowed_returns"` // List of allowed return actions
}

// VCLVariable represents a VCL variable with its type and access permissions
type VCLVariable struct {
	Type          string   `json:"type"`           // VCL type name
	ReadableFrom  []string `json:"readable_from"`  // VCL methods where variable can be read
	WritableFrom  []string `json:"writable_from"`  // VCL methods where variable can be written
	UnsetableFrom []string `json:"unsetable_from"` // VCL methods where variable can be unset
	VersionLow    int      `json:"version_low"`    // Minimum VCL version
	VersionHigh   int      `json:"version_high"`   // Maximum VCL version
}

// VCLType represents a VCL type definition
type VCLType struct {
	CType    string `json:"c_type"`   // Corresponding C type
	Internal bool   `json:"internal"` // Whether this is an internal type
}

// StorageVariable represents a storage-engine specific variable
type StorageVariable struct {
	Name        string `json:"name"`        // Variable name
	Type        string `json:"type"`        // VCL type name
	Default     string `json:"default"`     // Default value
	Description string `json:"description"` // Short description
	Docstring   string `json:"docstring"`   // Detailed documentation
}

// ContextType represents the execution context for VCL methods
type ContextType string

const (
	ClientContext       ContextType = "C" // Client-side methods
	BackendContext      ContextType = "B" // Backend-side methods
	HousekeepingContext ContextType = "H" // Housekeeping methods
)

// String returns the string representation of the context type
func (c ContextType) String() string {
	switch c {
	case ClientContext:
		return "Client"
	case BackendContext:
		return "Backend"
	case HousekeepingContext:
		return "Housekeeping"
	default:
		return "Unknown"
	}
}

// IsValidReturnAction checks if a return action is valid for a given method
func (m *VCLMethod) IsValidReturnAction(action string) bool {
	for _, allowed := range m.AllowedReturns {
		if allowed == action {
			return true
		}
	}
	return false
}

// IsReadableInMethod checks if a variable is readable in a given method
func (v *VCLVariable) IsReadableInMethod(method string, methods map[string]VCLMethod) bool {
	return v.isAccessibleInMethod(method, v.ReadableFrom, methods)
}

// IsWritableInMethod checks if a variable is writable in a given method
func (v *VCLVariable) IsWritableInMethod(method string, methods map[string]VCLMethod) bool {
	return v.isAccessibleInMethod(method, v.WritableFrom, methods)
}

// IsUnsetableInMethod checks if a variable is unsetable in a given method
func (v *VCLVariable) IsUnsetableInMethod(method string, methods map[string]VCLMethod) bool {
	return v.isAccessibleInMethod(method, v.UnsetableFrom, methods)
}

// IsAvailableInVersion checks if a variable is available in a specific VCL version
func (v *VCLVariable) IsAvailableInVersion(version int) bool {
	return version >= v.VersionLow && version <= v.VersionHigh
}

// isAccessibleInMethod is a helper that resolves context permissions to specific methods
func (v *VCLVariable) isAccessibleInMethod(method string, permissions []string, methods map[string]VCLMethod) bool {
	for _, permission := range permissions {
		switch permission {
		case "all":
			return true
		case "client":
			if methodInfo, exists := methods[method]; exists && methodInfo.Context == string(ClientContext) {
				return true
			}
		case "backend":
			if methodInfo, exists := methods[method]; exists && methodInfo.Context == string(BackendContext) {
				return true
			}
		case "both":
			if methodInfo, exists := methods[method]; exists &&
				(methodInfo.Context == string(ClientContext) || methodInfo.Context == string(BackendContext)) {
				return true
			}
		default:
			// Direct method name match
			if permission == method {
				return true
			}
			// Also try with vcl_ prefix
			if permission == "vcl_"+method {
				return true
			}
			// And try without vcl_ prefix
			if strings.HasPrefix(permission, "vcl_") && strings.TrimPrefix(permission, "vcl_") == method {
				return true
			}
		}
	}
	return false
}
