package metadata

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed metadata.json
var embeddedMetadata []byte

// MetadataLoader handles loading and caching VCL metadata
type MetadataLoader struct {
	metadata *VCLMetadata
	mu       sync.RWMutex
}

// New creates a new metadata instance
func New() *MetadataLoader {
	var metadata VCLMetadata
	if err := json.Unmarshal(embeddedMetadata, &metadata); err != nil {
		panic("failed to parse embedded metadata: " + err.Error() + "")
	}
	return &MetadataLoader{
		metadata: &metadata,
	}
}

// GetMetadata returns the loaded metadata (thread-safe)
func (ml *MetadataLoader) GetMetadata() (*VCLMetadata, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.metadata == nil {
		return nil, fmt.Errorf("metadata not available - failed to initialize embedded metadata")
	}

	return ml.metadata, nil
}

// GetMethods returns the VCL methods metadata
func (ml *MetadataLoader) GetMethods() (map[string]VCLMethod, error) {
	metadata, err := ml.GetMetadata()
	if err != nil {
		return nil, err
	}
	return metadata.VCLMethods, nil
}

// GetVariables returns the VCL variables metadata
func (ml *MetadataLoader) GetVariables() (map[string]VCLVariable, error) {
	metadata, err := ml.GetMetadata()
	if err != nil {
		return nil, err
	}
	return metadata.VCLVariables, nil
}

// GetTypes returns the VCL types metadata
func (ml *MetadataLoader) GetTypes() (map[string]VCLType, error) {
	metadata, err := ml.GetMetadata()
	if err != nil {
		return nil, err
	}
	return metadata.VCLTypes, nil
}

// GetTokens returns the VCL tokens metadata
func (ml *MetadataLoader) GetTokens() (map[string]string, error) {
	metadata, err := ml.GetMetadata()
	if err != nil {
		return nil, err
	}
	return metadata.VCLTokens, nil
}

// GetStorageVariables returns the storage variables metadata
func (ml *MetadataLoader) GetStorageVariables() ([]StorageVariable, error) {
	metadata, err := ml.GetMetadata()
	if err != nil {
		return nil, err
	}
	return metadata.StorageVariables, nil
}

// ValidateReturnAction checks if a return action is valid for a given method
func (ml *MetadataLoader) ValidateReturnAction(method, action string) error {
	methods, err := ml.GetMethods()
	if err != nil {
		return err
	}

	methodInfo, exists := methods[method]
	if !exists {
		return fmt.Errorf("unknown VCL method: %s", method)
	}

	if !methodInfo.IsValidReturnAction(action) {
		return fmt.Errorf("return action '%s' is not allowed in method '%s'. Allowed actions: %v",
			action, method, methodInfo.AllowedReturns)
	}

	return nil
}

// normalizeDynamicVariable converts dynamic variables to their pattern forms
func normalizeDynamicVariable(variable string) string {
	// Handle HTTP header patterns: req.http.*, bereq.http.*, beresp.http.*, resp.http.*, obj.http.*
	if strings.Contains(variable, ".http.") {
		parts := strings.Split(variable, ".http.")
		if len(parts) == 2 {
			return parts[0] + ".http."
		}
	}

	// Handle storage.<name>.* patterns
	if strings.HasPrefix(variable, "storage.") {
		parts := strings.Split(variable, ".")
		if len(parts) >= 3 {
			// storage.<name>.property -> normalize to pattern if it exists
			// For now, we'll skip storage validation as it's more complex
			return ""
		}
	}

	return ""
}

// ValidateVariableAccess checks if a variable access (read/write/unset) is valid in a method
func (ml *MetadataLoader) ValidateVariableAccess(variable, method, accessType string) error {
	variables, err := ml.GetVariables()
	if err != nil {
		return err
	}

	methods, err := ml.GetMethods()
	if err != nil {
		return err
	}

	varInfo, exists := variables[variable]
	if !exists {
		// Try to match dynamic patterns
		normalizedVar := normalizeDynamicVariable(variable)
		if normalizedVar != "" {
			varInfo, exists = variables[normalizedVar]
		}
		if !exists {
			return fmt.Errorf("unknown VCL variable: %s", variable)
		}
	}

	var isValid bool
	switch accessType {
	case "read":
		isValid = varInfo.IsReadableInMethod(method, methods)
	case "write":
		isValid = varInfo.IsWritableInMethod(method, methods)
	case "unset":
		isValid = varInfo.IsUnsetableInMethod(method, methods)
	default:
		return fmt.Errorf("invalid access type: %s (must be read, write, or unset)", accessType)
	}

	if !isValid {
		return fmt.Errorf("variable '%s' cannot be %s in method '%s'", variable, accessType+"d", method)
	}

	return nil
}

// GetMethodsForContext returns all methods for a given context (client/backend/housekeeping)
func (ml *MetadataLoader) GetMethodsForContext(context ContextType) ([]string, error) {
	methods, err := ml.GetMethods()
	if err != nil {
		return nil, err
	}

	var result []string
	for name, method := range methods {
		if method.Context == string(context) {
			result = append(result, name)
		}
	}

	return result, nil
}
