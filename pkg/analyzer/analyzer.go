package analyzer

import (
	"fmt"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/parser"
	"github.com/perbu/vclparser/pkg/types"
	"github.com/perbu/vclparser/pkg/vmod"
)

// Analyzer performs semantic analysis on VCL AST
type Analyzer struct {
	symbolTable       *types.SymbolTable
	vmodValidator     *VMODValidator
	returnValidator   *ReturnActionValidator
	variableValidator *VariableAccessValidator
	versionValidator  *VersionValidator
	metadataLoader    *metadata.MetadataLoader
	errors            []string
}

// NewAnalyzer creates a new semantic analyzer
func NewAnalyzer(registry *vmod.Registry) *Analyzer {
	symbolTable := types.NewSymbolTable()
	vmodValidator := NewVMODValidator(registry, symbolTable)

	// Load metadata for return action validation
	metadataLoader := metadata.New()

	returnValidator := NewReturnActionValidator(metadataLoader)
	variableValidator := NewVariableAccessValidator(metadataLoader, symbolTable)
	versionValidator := NewVersionValidator(metadataLoader)

	return &Analyzer{
		symbolTable:       symbolTable,
		vmodValidator:     vmodValidator,
		returnValidator:   returnValidator,
		variableValidator: variableValidator,
		versionValidator:  versionValidator,
		metadataLoader:    metadataLoader,
		errors:            []string{},
	}
}

// Analyze performs complete semantic analysis on an AST
func (a *Analyzer) Analyze(program *ast.Program) []string {
	a.errors = []string{}

	// Perform VMOD validation
	vmodErrors := a.vmodValidator.Validate(program)
	a.errors = append(a.errors, vmodErrors...)

	// Perform return action validation
	returnErrors := a.returnValidator.Validate(program)
	a.errors = append(a.errors, returnErrors...)

	// Perform variable access validation
	variableErrors := a.variableValidator.Validate(program)
	a.errors = append(a.errors, variableErrors...)

	// Perform VCL version compatibility validation
	versionErrors := a.versionValidator.Validate(program)
	a.errors = append(a.errors, versionErrors...)

	// TODO: Add other semantic analysis passes here
	// - Type checking
	// - Control flow analysis

	return a.errors
}

// AnalyzeWithSymbolTable performs complete semantic analysis on an AST and returns validation errors
// along with the populated symbol table. This is useful when external code needs access to the
// symbol table for additional processing or symbol lookups after validation.
func (a *Analyzer) AnalyzeWithSymbolTable(program *ast.Program) ([]string, *types.SymbolTable) {
	errors := a.Analyze(program)
	return errors, a.symbolTable
}

// GetSymbolTable returns the symbol table
func (a *Analyzer) GetSymbolTable() *types.SymbolTable {
	return a.symbolTable
}

// ValidateVCLFile validates a VCL file with VMOD support using the provided registry.
// This is a convenience function that creates an analyzer instance and performs complete
// semantic validation. Returns validation errors and an error if validation fails.
// Use this when you have a VMOD registry and want simple file validation.
// Returns a list of error messages and an error if validation fails.
func ValidateVCLFile(program *ast.Program, registry *vmod.Registry) ([]string, error) {
	analyzer := NewAnalyzer(registry)
	errors := analyzer.Analyze(program)
	if len(errors) > 0 {
		return errors, fmt.Errorf("validation failed with %d error(s)", len(errors))
	}
	return nil, nil
}

// ParseWithCustomVMODValidation parses VCL input and performs comprehensive semantic validation
// using a custom VMOD registry. This combines parsing and validation in a single operation,
// returning the AST, validation errors, and any parse errors. Useful when you need both
// parsing and validation with specific VMOD configurations in one step.
func ParseWithCustomVMODValidation(input, filename string, registry *vmod.Registry) (*ast.Program, []string, error) {
	// Parse the VCL code
	program, err := parser.Parse(input, filename)
	if err != nil {
		return program, nil, err
	}

	// Perform VMOD validation with the provided registry
	analyzer := NewAnalyzer(registry)
	validationErrors := analyzer.Analyze(program)

	return program, validationErrors, nil
}
