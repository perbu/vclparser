package analyzer

import (
	"fmt"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
)

// ReturnActionValidator validates return statements against VCL metadata
type ReturnActionValidator struct {
	loader        *metadata.MetadataLoader
	currentMethod string
	errors        []string
}

// NewReturnActionValidator creates a new return action validator
func NewReturnActionValidator(loader *metadata.MetadataLoader) *ReturnActionValidator {
	return &ReturnActionValidator{
		loader: loader,
		errors: []string{},
	}
}

// Validate validates all return statements in a VCL program
func (rav *ReturnActionValidator) Validate(program *ast.Program) []string {
	rav.errors = []string{}

	// Visit all subroutines and validate return statements
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok {
			rav.currentMethod = subDecl.Name
			rav.validateSubroutineReturns(subDecl)
		}
	}

	return rav.errors
}

// validateSubroutineReturns validates return statements in a subroutine
func (rav *ReturnActionValidator) validateSubroutineReturns(sub *ast.SubDecl) {
	// Only validate built-in VCL subroutines (those starting with vcl_)
	if !isBuiltinSubroutine(sub.Name) {
		return
	}

	// Remove vcl_ prefix for metadata lookup
	methodName := extractMethodName(sub.Name)

	// Find all return statements in the subroutine
	returnStmts := rav.findReturnStatements(sub.Body.Statements)

	for _, returnStmt := range returnStmts {
		if err := rav.validateReturnStatement(returnStmt, methodName); err != nil {
			rav.errors = append(rav.errors, err.Error())
		}
	}
}

// validateReturnStatement validates a single return statement
func (rav *ReturnActionValidator) validateReturnStatement(stmt *ast.ReturnStatement, methodName string) error {
	if stmt.Action == nil {
		// Empty return is always valid (used in custom subroutines)
		return nil
	}

	// Extract action name from the expression
	actionName, err := rav.extractActionName(stmt.Action)
	if err != nil {
		return fmt.Errorf("invalid return action at line %d: %v", stmt.StartPos.Line, err)
	}

	// Validate against metadata
	if err := rav.loader.ValidateReturnAction(methodName, actionName); err != nil {
		return fmt.Errorf("at line %d: %v", stmt.StartPos.Line, err)
	}

	return nil
}

// extractActionName extracts the action name from a return expression
func (rav *ReturnActionValidator) extractActionName(expr ast.Expression) (string, error) {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Name, nil
	case *ast.CallExpression:
		// Handle function calls like synth(200, "OK")
		if ident, ok := e.Function.(*ast.Identifier); ok {
			return ident.Name, nil
		}
		return "", fmt.Errorf("invalid function call in return statement")
	default:
		return "", fmt.Errorf("unsupported return action type: %T", expr)
	}
}

// findReturnStatements recursively finds all return statements in a statement list
func (rav *ReturnActionValidator) findReturnStatements(statements []ast.Statement) []*ast.ReturnStatement {
	var returns []*ast.ReturnStatement

	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *ast.ReturnStatement:
			returns = append(returns, s)
		case *ast.IfStatement:
			// Check then branch
			if s.Then != nil {
				if blockStmt, ok := s.Then.(*ast.BlockStatement); ok {
					returns = append(returns, rav.findReturnStatements(blockStmt.Statements)...)
				} else if returnStmt, ok := s.Then.(*ast.ReturnStatement); ok {
					returns = append(returns, returnStmt)
				}
			}
			// Check else branch if it exists
			if s.Else != nil {
				if blockStmt, ok := s.Else.(*ast.BlockStatement); ok {
					returns = append(returns, rav.findReturnStatements(blockStmt.Statements)...)
				} else if returnStmt, ok := s.Else.(*ast.ReturnStatement); ok {
					returns = append(returns, returnStmt)
				}
			}
		case *ast.BlockStatement:
			returns = append(returns, rav.findReturnStatements(s.Statements)...)
		}
	}

	return returns
}

// isBuiltinSubroutine checks if a subroutine is a built-in VCL subroutine
func isBuiltinSubroutine(name string) bool {
	return len(name) > 4 && name[:4] == "vcl_"
}

// extractMethodName removes the vcl_ prefix from a subroutine name
func extractMethodName(subroutineName string) string {
	if len(subroutineName) > 4 && subroutineName[:4] == "vcl_" {
		return subroutineName[4:]
	}
	return subroutineName
}

// ValidateReturnActions is a convenience function to validate return actions in a program
func ValidateReturnActions(program *ast.Program, loader *metadata.MetadataLoader) ([]string, error) {
	validator := NewReturnActionValidator(loader)
	errors := validator.Validate(program)

	if len(errors) > 0 {
		return errors, fmt.Errorf("found %d return action validation error(s)", len(errors))
	}

	return nil, nil
}
