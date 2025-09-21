package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// MetadataLoader handles loading and caching VCL metadata
type MetadataLoader struct {
	metadata *VCLMetadata
	mu       sync.RWMutex
}

// NewMetadataLoader creates a new metadata loader
func NewMetadataLoader() *MetadataLoader {
	return &MetadataLoader{}
}

// LoadFromFile loads metadata from a JSON file
func (ml *MetadataLoader) LoadFromFile(filepath string) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read metadata file %s: %w", filepath, err)
	}

	var metadata VCLMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata JSON: %w", err)
	}

	ml.metadata = &metadata
	return nil
}

// LoadDefault loads metadata from embedded data
func (ml *MetadataLoader) LoadDefault() error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	var metadata VCLMetadata
	if err := json.Unmarshal(embeddedMetadata, &metadata); err != nil {
		return fmt.Errorf("failed to parse embedded metadata: %w", err)
	}

	ml.metadata = &metadata
	return nil
}

// GetMetadata returns the loaded metadata (thread-safe)
func (ml *MetadataLoader) GetMetadata() (*VCLMetadata, error) {
	ml.mu.RLock()
	defer ml.mu.RUnlock()

	if ml.metadata == nil {
		return nil, fmt.Errorf("metadata not loaded - call LoadFromFile or LoadDefault first")
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

// Global instance for convenience
var DefaultLoader = NewMetadataLoader()

// LoadDefault loads metadata using the default loader
func LoadDefault() error {
	return DefaultLoader.LoadDefault()
}

// GetMetadata returns metadata using the default loader
func GetMetadata() (*VCLMetadata, error) {
	return DefaultLoader.GetMetadata()
}
