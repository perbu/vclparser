package analyzer

import (
	"fmt"
	"strings"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/types"
)

// VariableAccessValidator validates variable access permissions against VCL metadata
type VariableAccessValidator struct {
	loader        *metadata.MetadataLoader
	symbolTable   *types.SymbolTable
	currentMethod string
	errors        []string
}

// NewVariableAccessValidator creates a new variable access validator
func NewVariableAccessValidator(loader *metadata.MetadataLoader, symbolTable *types.SymbolTable) *VariableAccessValidator {
	return &VariableAccessValidator{
		loader:      loader,
		symbolTable: symbolTable,
		errors:      []string{},
	}
}

// Validate validates all variable accesses in a VCL program
func (vav *VariableAccessValidator) Validate(program *ast.Program) []string {
	vav.errors = []string{}

	// Visit all subroutines and validate variable accesses
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok {
			vav.currentMethod = extractMethodName(subDecl.Name)
			vav.validateSubroutineVariableAccess(subDecl)
		}
	}

	return vav.errors
}

// validateSubroutineVariableAccess validates variable accesses in a subroutine
func (vav *VariableAccessValidator) validateSubroutineVariableAccess(sub *ast.SubDecl) {
	// Only validate built-in VCL subroutines
	if !isBuiltinSubroutine(sub.Name) {
		return
	}

	// Walk the AST and find variable accesses
	vav.walkStatements(sub.Body.Statements)
}

// walkStatements recursively walks statement AST nodes to find variable accesses
func (vav *VariableAccessValidator) walkStatements(statements []ast.Statement) {
	for _, stmt := range statements {
		vav.walkStatement(stmt)
	}
}

// walkStatement walks a single statement to find variable accesses
func (vav *VariableAccessValidator) walkStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.SetStatement:
		// Variable assignment - validate write access
		varName := vav.extractVariableName(s.Variable)
		if varName != "" {
			if err := vav.validateVariableAccess(varName, "write", s.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}
		// Also validate read access to the value expression
		vav.walkExpression(s.Value)

	case *ast.UnsetStatement:
		// Variable unset - validate unset access
		varName := vav.extractVariableName(s.Variable)
		if varName != "" {
			if err := vav.validateVariableAccess(varName, "unset", s.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}

	case *ast.IfStatement:
		// Validate condition expression
		vav.walkExpression(s.Condition)
		// Walk then branch
		if s.Then != nil {
			if blockStmt, ok := s.Then.(*ast.BlockStatement); ok {
				vav.walkStatements(blockStmt.Statements)
			} else {
				vav.walkStatement(s.Then)
			}
		}
		// Walk else branch
		if s.Else != nil {
			if blockStmt, ok := s.Else.(*ast.BlockStatement); ok {
				vav.walkStatements(blockStmt.Statements)
			} else {
				vav.walkStatement(s.Else)
			}
		}

	case *ast.BlockStatement:
		vav.walkStatements(s.Statements)

	case *ast.ExpressionStatement:
		vav.walkExpression(s.Expression)

	case *ast.CallStatement:
		vav.walkExpression(s.Function)

	case *ast.ReturnStatement:
		if s.Action != nil {
			vav.walkExpression(s.Action)
		}

	case *ast.SyntheticStatement:
		vav.walkExpression(s.Response)

	case *ast.ErrorStatement:
		if s.Code != nil {
			vav.walkExpression(s.Code)
		}
		if s.Response != nil {
			vav.walkExpression(s.Response)
		}
	}
}

// walkExpression walks expression AST nodes to find variable reads
func (vav *VariableAccessValidator) walkExpression(expr ast.Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.Identifier:
		// Simple variable read - but skip if it's a return action, built-in function, or backend
		if !vav.isReturnActionOrBuiltin(e.Name) && !vav.isBackendOrVMODObject(e.Name) {
			if err := vav.validateVariableAccess(e.Name, "read", e.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}

	case *ast.MemberExpression:
		// Skip if this is a VMOD function/method call
		if vav.isVMODAccess(e) {
			return
		}
		// Member access like req.url, req.http.host
		varName := vav.extractMemberVariableName(e)
		if varName != "" {
			if err := vav.validateVariableAccess(varName, "read", e.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}

	case *ast.CallExpression:
		// Function call - validate arguments
		vav.walkExpression(e.Function)
		for _, arg := range e.Arguments {
			vav.walkExpression(arg)
		}
		for _, arg := range e.NamedArguments {
			vav.walkExpression(arg)
		}

	case *ast.BinaryExpression:
		vav.walkExpression(e.Left)
		vav.walkExpression(e.Right)

	case *ast.UnaryExpression:
		vav.walkExpression(e.Operand)

	case *ast.ParenthesizedExpression:
		vav.walkExpression(e.Expression)

	case *ast.RegexMatchExpression:
		vav.walkExpression(e.Left)
		vav.walkExpression(e.Right)

	case *ast.AssignmentExpression:
		// Validate write access to left side
		varName := vav.extractVariableName(e.Left)
		if varName != "" {
			if err := vav.validateVariableAccess(varName, "write", e.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}
		// Validate read access to right side
		vav.walkExpression(e.Right)

	case *ast.IndexExpression:
		vav.walkExpression(e.Object)
		vav.walkExpression(e.Index)

	case *ast.UpdateExpression:
		// Increment/decrement operations require both read and write access
		varName := vav.extractVariableName(e.Operand)
		if varName != "" {
			if err := vav.validateVariableAccess(varName, "read", e.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
			if err := vav.validateVariableAccess(varName, "write", e.StartPos.Line); err != nil {
				vav.errors = append(vav.errors, err.Error())
			}
		}

	// Literal expressions don't need validation
	case *ast.StringLiteral, *ast.IntegerLiteral, *ast.FloatLiteral, *ast.BooleanLiteral:
		// No validation needed for literals
	}
}

// isVMODAccess checks if an expression is a VMOD function call or object method
func (vav *VariableAccessValidator) isVMODAccess(expr ast.Expression) bool {
	if expr == nil {
		return false
	}

	switch e := expr.(type) {
	case *ast.MemberExpression:
		// Get the base identifier from the member expression
		base := vav.getBaseIdentifier(e)
		if base != "" {
			// Check if it's an imported VMOD module
			if vav.symbolTable.IsModuleImported(base) {
				return true
			}
			// Check if it's a VMOD object instance
			symbol := vav.symbolTable.Lookup(base)
			if symbol != nil && symbol.Kind == types.SymbolVMODObject {
				return true
			}
		}
	case *ast.Identifier:
		// Direct module reference
		return vav.symbolTable.IsModuleImported(e.Name)
	}
	return false
}

// isBackendOrVMODObject checks if an identifier refers to a backend or VMOD object
func (vav *VariableAccessValidator) isBackendOrVMODObject(name string) bool {
	symbol := vav.symbolTable.Lookup(name)
	if symbol != nil {
		return symbol.Kind == types.SymbolBackend || symbol.Kind == types.SymbolVMODObject
	}
	return false
}

// getBaseIdentifier extracts the base identifier from a member expression chain
func (vav *VariableAccessValidator) getBaseIdentifier(expr *ast.MemberExpression) string {
	current := expr
	for current != nil {
		if memberObj, ok := current.Object.(*ast.MemberExpression); ok {
			current = memberObj
		} else if ident, ok := current.Object.(*ast.Identifier); ok {
			return ident.Name
		} else {
			return ""
		}
	}
	return ""
}

// extractVariableName extracts variable name from an expression
func (vav *VariableAccessValidator) extractVariableName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Name
	case *ast.MemberExpression:
		return vav.extractMemberVariableName(e)
	default:
		return ""
	}
}

// extractMemberVariableName extracts full variable name from member expression
func (vav *VariableAccessValidator) extractMemberVariableName(expr *ast.MemberExpression) string {
	var parts []string

	// Walk the member chain to build the full variable name
	current := expr
	for current != nil {
		// Get the property name
		if prop, ok := current.Property.(*ast.Identifier); ok {
			parts = append([]string{prop.Name}, parts...) // Prepend to maintain order
		} else {
			return "" // Complex property access, can't validate
		}

		// Check if the object is another member expression
		if memberObj, ok := current.Object.(*ast.MemberExpression); ok {
			current = memberObj
		} else if ident, ok := current.Object.(*ast.Identifier); ok {
			parts = append([]string{ident.Name}, parts...) // Prepend base identifier
			break
		} else {
			return "" // Complex object access, can't validate
		}
	}

	return strings.Join(parts, ".")
}

// validateVariableAccess validates variable access against metadata
func (vav *VariableAccessValidator) validateVariableAccess(varName, accessType string, line int) error {
	if err := vav.loader.ValidateVariableAccess(varName, vav.currentMethod, accessType); err != nil {
		return fmt.Errorf("at line %d: %v", line, err)
	}
	return nil
}

// isReturnActionOrBuiltin checks if an identifier is a return action or built-in function
func (vav *VariableAccessValidator) isReturnActionOrBuiltin(name string) bool {
	// Common return actions
	returnActions := map[string]bool{
		"hash": true, "lookup": true, "pass": true, "pipe": true, "fail": true,
		"synth": true, "restart": true, "deliver": true, "fetch": true,
		"abandon": true, "error": true, "retry": true, "miss": true,
		"ok": true, "connect": true, "vcl": true, "purge": true,
	}

	// Built-in functions and keywords
	builtins := map[string]bool{
		"true": true, "false": true, "ban": true, "call": true,
		"hash_data": true, "synthetic": true, "new": true,
		"set": true, "unset": true, "return": true, "if": true,
		"else": true, "elsif": true, "sub": true, "vcl": true,
	}

	return returnActions[name] || builtins[name]
}

// ValidateVariableAccesses is a convenience function to validate variable accesses in a program
func ValidateVariableAccesses(program *ast.Program, loader *metadata.MetadataLoader) ([]string, error) {
	symbolTable := types.NewSymbolTable()
	validator := NewVariableAccessValidator(loader, symbolTable)
	errors := validator.Validate(program)

	if len(errors) > 0 {
		return errors, fmt.Errorf("found %d variable access validation error(s)", len(errors))
	}

	return nil, nil
}
