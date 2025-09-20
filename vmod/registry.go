package vmod

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/varnish/vclparser/vcc"
)

// Registry manages VMOD definitions loaded from VCC files
type Registry struct {
	modules map[string]*vcc.Module
	mutex   sync.RWMutex
}

// NewRegistry creates a new VMOD registry
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]*vcc.Module),
	}
}

// LoadVCCDirectory loads all VCC files from a directory
func (r *Registry) LoadVCCDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .vcc files
		if !strings.HasSuffix(strings.ToLower(path), ".vcc") {
			return nil
		}

		if err := r.LoadVCCFile(path); err != nil {
			// Return error immediately instead of just logging warning
			return fmt.Errorf("failed to load VCC file %s: %v", path, err)
		}

		return nil
	})
}

// LoadVCCFile loads a single VCC file
func (r *Registry) LoadVCCFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open VCC file %s: %v", filename, err)
	}
	defer func() {
		_ = file.Close() // Ignore error in defer
	}()

	parser := vcc.NewParser(file)
	module, err := parser.Parse()
	if err != nil {
		return fmt.Errorf("failed to parse VCC file %s: %v", filename, err)
	}

	// Register the module
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if module.Name != "" {
		r.modules[module.Name] = module
	} else {
		return fmt.Errorf("module in %s has no name", filename)
	}

	return nil
}

// GetModule returns a module by name
func (r *Registry) GetModule(name string) (*vcc.Module, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	module, exists := r.modules[name]
	return module, exists
}

// ListModules returns a list of all registered module names
func (r *Registry) ListModules() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	return names
}

// GetFunction finds a function in a specific module
func (r *Registry) GetFunction(moduleName, functionName string) (*vcc.Function, error) {
	module, exists := r.GetModule(moduleName)
	if !exists {
		return nil, fmt.Errorf("module %s not found", moduleName)
	}
	// module is guaranteed non-nil when exists is true
	//nolint:nilaway
	function := module.FindFunction(functionName)
	if function == nil {
		return nil, fmt.Errorf("function %s not found in module %s", functionName, moduleName)
	}

	return function, nil
}

// GetObject finds an object in a specific module
func (r *Registry) GetObject(moduleName, objectName string) (*vcc.Object, error) {
	module, exists := r.GetModule(moduleName)
	if !exists {
		return nil, fmt.Errorf("module %s not found", moduleName)
	}
	// module is guaranteed non-nil when exists is true
	//nolint:nilaway
	object := module.FindObject(objectName)
	if object == nil {
		return nil, fmt.Errorf("object %s not found in module %s", objectName, moduleName)
	}

	return object, nil
}

// GetMethod finds a method on an object in a specific module
func (r *Registry) GetMethod(moduleName, objectName, methodName string) (*vcc.Method, error) {
	object, err := r.GetObject(moduleName, objectName)
	if err != nil {
		return nil, err
	}

	method := object.FindMethod(methodName)
	if method == nil {
		return nil, fmt.Errorf("method %s not found on object %s in module %s", methodName, objectName, moduleName)
	}

	return method, nil
}

// ValidateImport validates that a module exists and can be imported
func (r *Registry) ValidateImport(moduleName string) error {
	_, exists := r.GetModule(moduleName)
	if !exists {
		return fmt.Errorf("module %s is not available", moduleName)
	}
	return nil
}

// ValidateFunctionCall validates a VMOD function call
func (r *Registry) ValidateFunctionCall(moduleName, functionName string, argTypes []vcc.VCCType) error {
	function, err := r.GetFunction(moduleName, functionName)
	if err != nil {
		return err
	}

	return function.ValidateCall(argTypes)
}

// ValidateMethodCall validates a VMOD method call
func (r *Registry) ValidateMethodCall(moduleName, objectName, methodName string, argTypes []vcc.VCCType) error {
	method, err := r.GetMethod(moduleName, objectName, methodName)
	if err != nil {
		return err
	}

	return method.ValidateCall(argTypes)
}

// ValidateObjectConstruction validates object instantiation
func (r *Registry) ValidateObjectConstruction(moduleName, objectName string, argTypes []vcc.VCCType) error {
	object, err := r.GetObject(moduleName, objectName)
	if err != nil {
		return err
	}

	return object.ValidateConstruction(argTypes)
}

// GetModuleStats returns statistics about loaded modules
func (r *Registry) GetModuleStats() map[string]ModuleStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	stats := make(map[string]ModuleStats)
	for name, module := range r.modules {
		stats[name] = ModuleStats{
			Name:          name,
			Version:       module.Version,
			FunctionCount: len(module.Functions),
			ObjectCount:   len(module.Objects),
			EventCount:    len(module.Events),
			ABI:           module.ABI,
		}
	}

	return stats
}

// ModuleStats contains statistics about a module
type ModuleStats struct {
	Name          string
	Version       int
	FunctionCount int
	ObjectCount   int
	EventCount    int
	ABI           string
}

// String returns a string representation of module stats
func (ms ModuleStats) String() string {
	return fmt.Sprintf("%s v%d: %d functions, %d objects, %d events (ABI: %s)",
		ms.Name, ms.Version, ms.FunctionCount, ms.ObjectCount, ms.EventCount, ms.ABI)
}

// Clear removes all modules from the registry
func (r *Registry) Clear() {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.modules = make(map[string]*vcc.Module)
}

// ModuleExists checks if a module is registered
func (r *Registry) ModuleExists(name string) bool {
	_, exists := r.GetModule(name)
	return exists
}

// GetBuiltinModules returns a list of commonly available builtin modules
func (r *Registry) GetBuiltinModules() []string {
	builtins := []string{"std", "directors"}
	var available []string

	for _, name := range builtins {
		if r.ModuleExists(name) {
			available = append(available, name)
		}
	}

	return available
}

// DefaultRegistry is a global registry instance
var DefaultRegistry = NewRegistry()

// LoadDefaultVCCFiles loads VCC files from the default vcclib directory
func LoadDefaultVCCFiles() error {
	// Try to find vcclib directory in multiple locations
	possibleDirs := []string{
		"vcclib",       // Relative to current directory
		"./vcclib",     // Explicit relative path
		"../vcclib",    // One level up
		"../../vcclib", // Two levels up (for nested test directories)
	}

	for _, vccDir := range possibleDirs {
		if _, err := os.Stat(vccDir); err == nil {
			// Found the directory, try to load it
			if err := DefaultRegistry.LoadVCCDirectory(vccDir); err != nil {
				// Log the error but continue trying other directories
				continue
			}
			return nil // Successfully loaded
		}
	}

	// If not found in any location, return an error but don't fail completely
	return fmt.Errorf("vcclib directory not found in any of the expected locations: %v", possibleDirs)
}

// Init initializes the default registry
func Init() error {
	return LoadDefaultVCCFiles()
}
