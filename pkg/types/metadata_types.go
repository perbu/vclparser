package types

import (
	"fmt"
	"sync"

	"github.com/perbu/vclparser/pkg/metadata"
)

// MetadataTypeSystem provides types from the VCL metadata
type MetadataTypeSystem struct {
	loader    *metadata.MetadataLoader
	typeCache map[string]*BasicType
	mu        sync.RWMutex
}

// NewMetadataTypeSystem creates a new metadata-driven type system
func NewMetadataTypeSystem(loader *metadata.MetadataLoader) *MetadataTypeSystem {
	return &MetadataTypeSystem{
		loader:    loader,
		typeCache: make(map[string]*BasicType),
	}
}

// LoadTypes initializes the type system from metadata
func (mts *MetadataTypeSystem) LoadTypes() error {
	mts.mu.Lock()
	defer mts.mu.Unlock()

	types, err := mts.loader.GetTypes()
	if err != nil {
		return fmt.Errorf("failed to load types from metadata: %w", err)
	}

	// Clear existing cache
	mts.typeCache = make(map[string]*BasicType)

	// Create BasicType instances for all metadata types
	for typeName, typeInfo := range types {
		// Skip internal types for now
		if typeInfo.Internal {
			continue
		}

		mts.typeCache[typeName] = &BasicType{
			Name: typeName,
		}
	}

	return nil
}

// GetType returns a type by name, loading it from metadata if necessary
func (mts *MetadataTypeSystem) GetType(name string) (Type, error) {
	if mts == nil {
		return nil, fmt.Errorf("MetadataTypeSystem is nil")
	}
	mts.mu.RLock()
	if typ, exists := mts.typeCache[name]; exists {
		mts.mu.RUnlock()
		return typ, nil
	}
	mts.mu.RUnlock()

	// Type not in cache, try to load it
	types, err := mts.loader.GetTypes()
	if err != nil {
		return nil, fmt.Errorf("failed to load types from metadata: %w", err)
	}

	typeInfo, exists := types[name]
	if !exists {
		return nil, fmt.Errorf("unknown VCL type: %s", name)
	}

	if typeInfo.Internal {
		return nil, fmt.Errorf("type %s is internal and not accessible in VCL", name)
	}

	mts.mu.Lock()
	defer mts.mu.Unlock()

	// Double-check cache again (race condition protection)
	if typ, exists := mts.typeCache[name]; exists {
		return typ, nil
	}

	// Create and cache the type
	typ := &BasicType{Name: name}
	mts.typeCache[name] = typ
	return typ, nil
}

// GetAllTypes returns all available types
func (mts *MetadataTypeSystem) GetAllTypes() (map[string]Type, error) {
	err := mts.LoadTypes()
	if err != nil {
		return nil, err
	}

	mts.mu.RLock()
	defer mts.mu.RUnlock()

	result := make(map[string]Type)
	for name, typ := range mts.typeCache {
		result[name] = typ
	}

	return result, nil
}

// IsValidType checks if a type name is valid according to metadata
func (mts *MetadataTypeSystem) IsValidType(name string) bool {
	_, err := mts.GetType(name)
	return err == nil
}

// GetCType returns the corresponding C type for a VCL type
func (mts *MetadataTypeSystem) GetCType(vclType string) (string, error) {
	types, err := mts.loader.GetTypes()
	if err != nil {
		return "", fmt.Errorf("failed to load types from metadata: %w", err)
	}

	typeInfo, exists := types[vclType]
	if !exists {
		return "", fmt.Errorf("unknown VCL type: %s", vclType)
	}

	return typeInfo.CType, nil
}

// Global instance for backward compatibility
var DefaultMetadataTypeSystem *MetadataTypeSystem

// InitializeMetadataTypes initializes the global metadata type system
func InitializeMetadataTypes() error {
	loader := metadata.New()
	DefaultMetadataTypeSystem = NewMetadataTypeSystem(loader)
	return DefaultMetadataTypeSystem.LoadTypes()
}

// GetMetadataType returns a type using the default metadata type system
func GetMetadataType(name string) (Type, error) {
	if DefaultMetadataTypeSystem == nil {
		if err := InitializeMetadataTypes(); err != nil {
			return nil, err
		}
	}
	return DefaultMetadataTypeSystem.GetType(name)
}

// Enhanced type variables that use metadata
var (
	// Keep backward compatibility by providing common types
	MetadataString   Type
	MetadataInt      Type
	MetadataReal     Type
	MetadataBool     Type
	MetadataTime     Type
	MetadataDuration Type
	MetadataIP       Type
	MetadataBackend  Type
	MetadataACL      Type
	MetadataProbe    Type
	MetadataHeader   Type
	MetadataVoid     Type
	MetadataHTTP     Type
	MetadataBytes    Type
	MetadataBlob     Type
	MetadataBody     Type
)

// initializeCommonTypes loads common types into global variables
func initializeCommonTypes() error {
	if DefaultMetadataTypeSystem == nil {
		return fmt.Errorf("metadata type system not initialized")
	}

	var err error
	if MetadataString, err = DefaultMetadataTypeSystem.GetType("STRING"); err != nil {
		return err
	}
	if MetadataInt, err = DefaultMetadataTypeSystem.GetType("INT"); err != nil {
		return err
	}
	if MetadataReal, err = DefaultMetadataTypeSystem.GetType("REAL"); err != nil {
		return err
	}
	if MetadataBool, err = DefaultMetadataTypeSystem.GetType("BOOL"); err != nil {
		return err
	}
	if MetadataTime, err = DefaultMetadataTypeSystem.GetType("TIME"); err != nil {
		return err
	}
	if MetadataDuration, err = DefaultMetadataTypeSystem.GetType("DURATION"); err != nil {
		return err
	}
	if MetadataIP, err = DefaultMetadataTypeSystem.GetType("IP"); err != nil {
		return err
	}
	if MetadataBackend, err = DefaultMetadataTypeSystem.GetType("BACKEND"); err != nil {
		return err
	}
	if MetadataACL, err = DefaultMetadataTypeSystem.GetType("ACL"); err != nil {
		return err
	}
	if MetadataProbe, err = DefaultMetadataTypeSystem.GetType("PROBE"); err != nil {
		return err
	}
	if MetadataHeader, err = DefaultMetadataTypeSystem.GetType("HEADER"); err != nil {
		return err
	}
	if MetadataVoid, err = DefaultMetadataTypeSystem.GetType("VOID"); err != nil {
		return err
	}
	if MetadataHTTP, err = DefaultMetadataTypeSystem.GetType("HTTP"); err != nil {
		return err
	}
	if MetadataBytes, err = DefaultMetadataTypeSystem.GetType("BYTES"); err != nil {
		return err
	}
	if MetadataBlob, err = DefaultMetadataTypeSystem.GetType("BLOB"); err != nil {
		return err
	}
	if MetadataBody, err = DefaultMetadataTypeSystem.GetType("BODY"); err != nil {
		return err
	}

	return nil
}

// InitializeWithMetadata initializes the metadata type system and common types
func InitializeWithMetadata() error {
	if err := InitializeMetadataTypes(); err != nil {
		return err
	}
	return initializeCommonTypes()
}
