package types

import (
	"fmt"
	"strings"

	"github.com/perbu/vclparser/pkg/lexer"
	"github.com/perbu/vclparser/pkg/metadata"
)

// MetadataSymbolTable is a symbol table that loads built-in symbols from VCL metadata
type MetadataSymbolTable struct {
	*SymbolTable
	loader     *metadata.MetadataLoader
	typeSystem *MetadataTypeSystem
}

// NewMetadataSymbolTable creates a new symbol table using VCL metadata
func NewMetadataSymbolTable(loader *metadata.MetadataLoader, typeSystem *MetadataTypeSystem) *MetadataSymbolTable {
	// Create base symbol table without built-ins
	global := NewScope("global", nil)
	st := &SymbolTable{
		currentScope: global,
		globalScope:  global,
	}

	mst := &MetadataSymbolTable{
		SymbolTable: st,
		loader:      loader,
		typeSystem:  typeSystem,
	}

	return mst
}

// LoadBuiltinSymbols loads all built-in VCL symbols from metadata
func (mst *MetadataSymbolTable) LoadBuiltinSymbols() error {
	variables, err := mst.loader.GetVariables()
	if err != nil {
		return fmt.Errorf("failed to load variables from metadata: %w", err)
	}

	methods, err := mst.loader.GetMethods()
	if err != nil {
		return fmt.Errorf("failed to load methods from metadata: %w", err)
	}

	// Load all variables from metadata
	for varName, varInfo := range variables {
		if err := mst.defineMetadataVariable(varName, varInfo, methods); err != nil {
			return fmt.Errorf("failed to define variable %s: %w", varName, err)
		}
	}

	// Load storage variables
	storageVars, err := mst.loader.GetStorageVariables()
	if err != nil {
		return fmt.Errorf("failed to load storage variables: %w", err)
	}

	for _, storageVar := range storageVars {
		if err := mst.defineStorageVariable(storageVar); err != nil {
			return fmt.Errorf("failed to define storage variable %s: %w", storageVar.Name, err)
		}
	}

	return nil
}

// defineMetadataVariable creates a Symbol from VCL metadata variable info
func (mst *MetadataSymbolTable) defineMetadataVariable(name string, varInfo metadata.VCLVariable, methods map[string]metadata.VCLMethod) error {
	// Get the type from our type system
	typ, err := mst.typeSystem.GetType(varInfo.Type)
	if err != nil {
		// If type is not found, create a basic type for it
		typ = &BasicType{Name: varInfo.Type}
	}

	// Resolve method permissions
	readableMethods := mst.resolveMethodPermissions(varInfo.ReadableFrom, methods)
	writableMethods := mst.resolveMethodPermissions(varInfo.WritableFrom, methods)
	unsetableMethods := mst.resolveMethodPermissions(varInfo.UnsetableFrom, methods)

	symbol := &Symbol{
		Name:      name,
		Kind:      SymbolVariable,
		Type:      typ,
		Position:  lexer.Position{}, // No position for built-ins
		Readable:  len(readableMethods) > 0,
		Writable:  len(writableMethods) > 0,
		Unsetable: len(unsetableMethods) > 0,
		Methods:   readableMethods, // Use readable methods as default
	}

	return mst.Define(symbol)
}

// defineStorageVariable creates symbols for storage-specific variables
func (mst *MetadataSymbolTable) defineStorageVariable(storageVar metadata.StorageVariable) error {
	// Get the type from our type system
	typ, err := mst.typeSystem.GetType(storageVar.Type)
	if err != nil {
		typ = &BasicType{Name: storageVar.Type}
	}

	// Storage variables are typically readable from all contexts
	// but this should be verified against actual Varnish behavior
	symbol := &Symbol{
		Name:     fmt.Sprintf("storage.*.%s", storageVar.Name),
		Kind:     SymbolVariable,
		Type:     typ,
		Position: lexer.Position{},
		Readable: true,
		Writable: false,           // Storage variables are typically read-only
		Methods:  []string{"all"}, // Available in all methods
	}

	return mst.Define(symbol)
}

// resolveMethodPermissions converts metadata permission strings to specific method names
func (mst *MetadataSymbolTable) resolveMethodPermissions(permissions []string, methods map[string]metadata.VCLMethod) []string {
	var resolved []string
	seenMethods := make(map[string]bool)

	for _, permission := range permissions {
		switch permission {
		case "all":
			// Add all methods
			for methodName := range methods {
				if !seenMethods[methodName] {
					resolved = append(resolved, methodName)
					seenMethods[methodName] = true
				}
			}
		case "client":
			// Add all client-side methods
			for methodName, methodInfo := range methods {
				if methodInfo.Context == "C" && !seenMethods[methodName] {
					resolved = append(resolved, methodName)
					seenMethods[methodName] = true
				}
			}
		case "backend":
			// Add all backend-side methods
			for methodName, methodInfo := range methods {
				if methodInfo.Context == "B" && !seenMethods[methodName] {
					resolved = append(resolved, methodName)
					seenMethods[methodName] = true
				}
			}
		case "both":
			// Add all client and backend methods
			for methodName, methodInfo := range methods {
				if (methodInfo.Context == "C" || methodInfo.Context == "B") && !seenMethods[methodName] {
					resolved = append(resolved, methodName)
					seenMethods[methodName] = true
				}
			}
		default:
			// Direct method name or special method context
			if !seenMethods[permission] {
				resolved = append(resolved, permission)
				seenMethods[permission] = true
			}
		}
	}

	return resolved
}

// ValidateVariableAccess validates variable access in a specific method context
func (mst *MetadataSymbolTable) ValidateVariableAccess(variable, method, accessType string) error {
	return mst.loader.ValidateVariableAccess(variable, method, accessType)
}

// ValidateReturnAction validates return actions in a specific method context
func (mst *MetadataSymbolTable) ValidateReturnAction(method, action string) error {
	return mst.loader.ValidateReturnAction(method, action)
}

// LookupWithAccess looks up a variable and validates access permissions
func (mst *MetadataSymbolTable) LookupWithAccess(name, method, accessType string) (*Symbol, error) {
	symbol := mst.Lookup(name)
	if symbol == nil {
		// Check if this might be a dynamic variable (like req.http.*)
		if err := mst.handleDynamicVariable(name, method, accessType); err != nil {
			return nil, fmt.Errorf("variable %s not found", name)
		}
		// If dynamic variable handling succeeded, return a generic symbol
		return &Symbol{
			Name: name,
			Kind: SymbolVariable,
			Type: &BasicType{Name: "STRING"}, // Most dynamic variables are strings
		}, nil
	}

	// Validate access
	if err := mst.ValidateVariableAccess(name, method, accessType); err != nil {
		return nil, err
	}

	return symbol, nil
}

// handleDynamicVariable handles variables like req.http.*, beresp.http.*, etc.
func (mst *MetadataSymbolTable) handleDynamicVariable(name, method, accessType string) error {
	// Check for HTTP header patterns
	if strings.Contains(name, ".http.") {
		// Extract base variable (req, beresp, etc.)
		parts := strings.SplitN(name, ".http.", 2)
		if len(parts) == 2 {
			baseVar := parts[0] + ".http."
			return mst.ValidateVariableAccess(baseVar, method, accessType)
		}
	}

	// Check for storage variable patterns
	if strings.HasPrefix(name, "storage.") && strings.Count(name, ".") >= 2 {
		// storage.<name>.<property> pattern
		parts := strings.SplitAfter(name, ".")
		if len(parts) >= 3 {
			// Get the property name (last part)
			property := parts[len(parts)-1]
			storageVars, err := mst.loader.GetStorageVariables()
			if err != nil {
				return err
			}

			// Check if this property exists in storage variables
			for _, storageVar := range storageVars {
				if storageVar.Name == property {
					// Storage variables are generally readable from all contexts
					if accessType == "read" {
						return nil
					}
					return fmt.Errorf("storage variables are read-only")
				}
			}
		}
	}

	return fmt.Errorf("unknown variable pattern: %s", name)
}

// GetMethodContext returns the context (client/backend/housekeeping) for a method
func (mst *MetadataSymbolTable) GetMethodContext(method string) (metadata.ContextType, error) {
	methods, err := mst.loader.GetMethods()
	if err != nil {
		return "", err
	}

	methodInfo, exists := methods[method]
	if !exists {
		return "", fmt.Errorf("unknown method: %s", method)
	}

	return metadata.ContextType(methodInfo.Context), nil
}

// CreateDefault creates a default metadata symbol table with loaded symbols
func CreateDefault() (*MetadataSymbolTable, error) {
	loader := metadata.New()

	if err := InitializeMetadataTypes(); err != nil {
		return nil, fmt.Errorf("failed to initialize types: %w", err)
	}

	mst := NewMetadataSymbolTable(loader, DefaultMetadataTypeSystem)
	if err := mst.LoadBuiltinSymbols(); err != nil {
		return nil, fmt.Errorf("failed to load builtin symbols: %w", err)
	}

	return mst, nil
}
