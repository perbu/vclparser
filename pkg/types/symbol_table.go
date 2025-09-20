package types

import (
	"fmt"

	"github.com/varnish/vclparser/pkg/lexer"
)

// SymbolKind represents the kind of symbol
type SymbolKind int

const (
	SymbolVariable SymbolKind = iota
	SymbolFunction
	SymbolBackend
	SymbolACL
	SymbolProbe
	SymbolSubroutine
	SymbolModule
	SymbolVMODFunction
	SymbolVMODObject
)

func (sk SymbolKind) String() string {
	switch sk {
	case SymbolVariable:
		return "Variable"
	case SymbolFunction:
		return "Function"
	case SymbolBackend:
		return "Backend"
	case SymbolACL:
		return "ACL"
	case SymbolProbe:
		return "Probe"
	case SymbolSubroutine:
		return "Subroutine"
	case SymbolModule:
		return "Module"
	case SymbolVMODFunction:
		return "VMOD Function"
	case SymbolVMODObject:
		return "VMOD Object"
	default:
		return "Unknown"
	}
}

// Symbol represents a symbol in the symbol table
type Symbol struct {
	Name      string
	Kind      SymbolKind
	Type      Type
	Position  lexer.Position
	Scope     string
	Readable  bool
	Writable  bool
	Unsetable bool
	Methods   []string // VCL methods where this symbol is accessible

	// VMOD-specific metadata
	ModuleName  string   // For VMOD objects and functions
	ObjectType  string   // For VMOD objects, the object type name
	VMODMethods []string // Available methods on VMOD objects
}

func (s *Symbol) String() string {
	return fmt.Sprintf("%s %s: %s", s.Kind, s.Name, s.Type)
}

// Scope represents a lexical scope
type Scope struct {
	Name    string
	Parent  *Scope
	Symbols map[string]*Symbol
}

// NewScope creates a new scope
func NewScope(name string, parent *Scope) *Scope {
	return &Scope{
		Name:    name,
		Parent:  parent,
		Symbols: make(map[string]*Symbol),
	}
}

// Define adds a symbol to the scope
func (s *Scope) Define(symbol *Symbol) error {
	if _, exists := s.Symbols[symbol.Name]; exists {
		return fmt.Errorf("symbol %s already defined in scope %s", symbol.Name, s.Name)
	}
	s.Symbols[symbol.Name] = symbol
	symbol.Scope = s.Name
	return nil
}

// Lookup finds a symbol in this scope or parent scopes
func (s *Scope) Lookup(name string) *Symbol {
	if symbol, exists := s.Symbols[name]; exists {
		return symbol
	}
	if s.Parent != nil {
		return s.Parent.Lookup(name)
	}
	return nil
}

// SymbolTable manages symbols and scopes
type SymbolTable struct {
	currentScope *Scope
	globalScope  *Scope
}

// NewSymbolTable creates a new symbol table
func NewSymbolTable() *SymbolTable {
	global := NewScope("global", nil)
	st := &SymbolTable{
		currentScope: global,
		globalScope:  global,
	}

	// Define built-in symbols
	st.defineBuiltins()

	return st
}

// EnterScope creates and enters a new scope
func (st *SymbolTable) EnterScope(name string) {
	newScope := NewScope(name, st.currentScope)
	st.currentScope = newScope
}

// ExitScope exits the current scope
func (st *SymbolTable) ExitScope() {
	if st.currentScope.Parent != nil {
		st.currentScope = st.currentScope.Parent
	}
}

// Define adds a symbol to the current scope
func (st *SymbolTable) Define(symbol *Symbol) error {
	return st.currentScope.Define(symbol)
}

// Lookup finds a symbol in the current scope or parent scopes
func (st *SymbolTable) Lookup(name string) *Symbol {
	return st.currentScope.Lookup(name)
}

// CurrentScope returns the current scope name
func (st *SymbolTable) CurrentScope() string {
	return st.currentScope.Name
}

// defineBuiltins defines built-in VCL variables and functions
func (st *SymbolTable) defineBuiltins() {
	// Built-in HTTP objects
	_ = st.Define(&Symbol{
		Name:     "req",
		Kind:     SymbolVariable,
		Type:     HTTP,
		Readable: true,
		Writable: false,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:     "bereq",
		Kind:     SymbolVariable,
		Type:     HTTP,
		Readable: true,
		Writable: false,
		Methods:  []string{"backend", "backend_fetch", "backend_response", "backend_error"},
	})

	_ = st.Define(&Symbol{
		Name:     "resp",
		Kind:     SymbolVariable,
		Type:     HTTP,
		Readable: true,
		Writable: false,
		Methods:  []string{"deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:     "beresp",
		Kind:     SymbolVariable,
		Type:     HTTP,
		Readable: true,
		Writable: false,
		Methods:  []string{"backend_response", "backend_error"},
	})

	// Built-in request variables
	_ = st.Define(&Symbol{
		Name:     "req.method",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Writable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:     "req.url",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Writable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:     "req.proto",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Writable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:      "req.http",
		Kind:      SymbolVariable,
		Type:      Header,
		Readable:  true,
		Writable:  true,
		Unsetable: true,
		Methods:   []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	// Built-in response variables
	_ = st.Define(&Symbol{
		Name:     "resp.status",
		Kind:     SymbolVariable,
		Type:     Int,
		Readable: true,
		Writable: true,
		Methods:  []string{"deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:     "resp.reason",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Writable: true,
		Methods:  []string{"deliver", "synth"},
	})

	_ = st.Define(&Symbol{
		Name:      "resp.http",
		Kind:      SymbolVariable,
		Type:      Header,
		Readable:  true,
		Writable:  true,
		Unsetable: true,
		Methods:   []string{"deliver", "synth"},
	})

	// Built-in backend response variables
	_ = st.Define(&Symbol{
		Name:     "beresp.status",
		Kind:     SymbolVariable,
		Type:     Int,
		Readable: true,
		Writable: true,
		Methods:  []string{"backend_response", "backend_error"},
	})

	_ = st.Define(&Symbol{
		Name:     "beresp.reason",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Writable: true,
		Methods:  []string{"backend_response", "backend_error"},
	})

	_ = st.Define(&Symbol{
		Name:      "beresp.http",
		Kind:      SymbolVariable,
		Type:      Header,
		Readable:  true,
		Writable:  true,
		Unsetable: true,
		Methods:   []string{"backend_response", "backend_error"},
	})

	// Built-in object variables
	_ = st.Define(&Symbol{
		Name:     "obj.status",
		Kind:     SymbolVariable,
		Type:     Int,
		Readable: true,
		Methods:  []string{"hit", "deliver"},
	})

	_ = st.Define(&Symbol{
		Name:     "obj.reason",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Methods:  []string{"hit", "deliver"},
	})

	_ = st.Define(&Symbol{
		Name:     "obj.http",
		Kind:     SymbolVariable,
		Type:     Header,
		Readable: true,
		Methods:  []string{"hit", "deliver"},
	})

	// Built-in client variables
	_ = st.Define(&Symbol{
		Name:     "client.ip",
		Kind:     SymbolVariable,
		Type:     IP,
		Readable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"},
	})

	// Built-in server variables
	_ = st.Define(&Symbol{
		Name:     "server.ip",
		Kind:     SymbolVariable,
		Type:     IP,
		Readable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth", "backend_fetch", "backend_response", "backend_error"},
	})

	_ = st.Define(&Symbol{
		Name:     "server.hostname",
		Kind:     SymbolVariable,
		Type:     String,
		Readable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth", "backend_fetch", "backend_response", "backend_error"},
	})

	// Built-in time variables
	_ = st.Define(&Symbol{
		Name:     "now",
		Kind:     SymbolVariable,
		Type:     Time,
		Readable: true,
		Methods:  []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth", "backend_fetch", "backend_response", "backend_error"},
	})

	// Built-in functions
	_ = st.Define(&Symbol{
		Name: "hash_data",
		Kind: SymbolFunction,
		Type: &FunctionType{
			Parameters: []Type{String},
			ReturnType: Void,
		},
		Methods: []string{"hash"},
	})

	_ = st.Define(&Symbol{
		Name: "regsub",
		Kind: SymbolFunction,
		Type: &FunctionType{
			Parameters: []Type{String, String, String},
			ReturnType: String,
		},
		Methods: []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth", "backend_fetch", "backend_response", "backend_error"},
	})

	_ = st.Define(&Symbol{
		Name: "regsuball",
		Kind: SymbolFunction,
		Type: &FunctionType{
			Parameters: []Type{String, String, String},
			ReturnType: String,
		},
		Methods: []string{"recv", "pipe", "pass", "hash", "purge", "miss", "hit", "deliver", "synth", "backend_fetch", "backend_response", "backend_error"},
	})
}

// ValidateAccess checks if a symbol can be accessed in the given VCL method
func (st *SymbolTable) ValidateAccess(symbolName, method string, accessType string) error {
	symbol := st.Lookup(symbolName)
	if symbol == nil {
		return fmt.Errorf("undefined symbol: %s", symbolName)
	}

	// Check if symbol is accessible in this method
	accessible := false
	for _, m := range symbol.Methods {
		if m == method {
			accessible = true
			break
		}
	}

	if !accessible {
		return fmt.Errorf("symbol %s not accessible in method %s", symbolName, method)
	}

	// Check access type
	switch accessType {
	case "read":
		if !symbol.Readable {
			return fmt.Errorf("symbol %s is not readable", symbolName)
		}
	case "write":
		if !symbol.Writable {
			return fmt.Errorf("symbol %s is not writable", symbolName)
		}
	case "unset":
		if !symbol.Unsetable {
			return fmt.Errorf("symbol %s cannot be unset", symbolName)
		}
	}

	return nil
}

// DefineModule adds a VMOD module to the symbol table
func (st *SymbolTable) DefineModule(moduleName string) error {
	return st.Define(&Symbol{
		Name: moduleName,
		Kind: SymbolModule,
		Type: Module,
	})
}

// DefineVMODFunction adds a VMOD function to the symbol table
func (st *SymbolTable) DefineVMODFunction(moduleName, functionName string, returnType Type) error {
	fullName := moduleName + "." + functionName
	return st.Define(&Symbol{
		Name:       fullName,
		Kind:       SymbolVMODFunction,
		Type:       returnType,
		ModuleName: moduleName,
	})
}

// DefineVMODObject adds a VMOD object instance to the symbol table
func (st *SymbolTable) DefineVMODObject(objectName, moduleName, objectType string) error {
	return st.Define(&Symbol{
		Name:       objectName,
		Kind:       SymbolVMODObject,
		Type:       Object,
		Scope:      st.currentScope.Name,
		ModuleName: moduleName,
		ObjectType: objectType,
		// VMODMethods will be populated from VCC registry if needed
	})
}

// LookupVMODFunction looks up a VMOD function by module and function name
func (st *SymbolTable) LookupVMODFunction(moduleName, functionName string) *Symbol {
	fullName := moduleName + "." + functionName
	return st.Lookup(fullName)
}

// IsModuleImported checks if a VMOD module has been imported
func (st *SymbolTable) IsModuleImported(moduleName string) bool {
	symbol := st.Lookup(moduleName)
	return symbol != nil && symbol.Kind == SymbolModule
}
