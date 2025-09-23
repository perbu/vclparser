package analyzer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/metadata"
)

// VersionValidator validates VCL version compatibility against metadata
type VersionValidator struct {
	loader *metadata.MetadataLoader
	errors []string
}

// NewVersionValidator creates a new version validator
func NewVersionValidator(loader *metadata.MetadataLoader) *VersionValidator {
	return &VersionValidator{
		loader: loader,
		errors: []string{},
	}
}

// Validate validates version compatibility for all features used in a VCL program
func (vv *VersionValidator) Validate(program *ast.Program) []string {
	vv.errors = []string{}

	// Extract VCL version from program
	vclVersion := vv.extractVCLVersion(program)
	if vclVersion == 0 {
		// If no version is specified, assume 4.0 for compatibility
		vclVersion = 40
	}

	// Validate variable usage against version constraints
	vv.validateVariableVersions(program, vclVersion)

	return vv.errors
}

// extractVCLVersion extracts and parses the VCL version declaration from the program AST,
// converting version strings like "4.0" or "4.1" into metadata format integers (40, 41).
// Returns 0 if no version is specified, enabling appropriate default handling.
func (vv *VersionValidator) extractVCLVersion(program *ast.Program) int {
	if program.VCLVersion == nil {
		return 0 // No version specified
	}

	version := program.VCLVersion.Version

	// Handle common version formats: "4.0", "4.1", etc.
	parts := strings.Split(version, ".")
	if len(parts) != 2 {
		vv.addError(fmt.Sprintf("invalid VCL version format: %s", version))
		return 0
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		vv.addError(fmt.Sprintf("invalid VCL major version: %s", parts[0]))
		return 0
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		vv.addError(fmt.Sprintf("invalid VCL minor version: %s", parts[1]))
		return 0
	}

	// Convert to metadata format (40 for 4.0, 41 for 4.1)
	return major*10 + minor
}

// validateVariableVersions performs comprehensive version compatibility checking for all variable
// accesses in the program against the specified VCL version. Validates that variables are available
// in the target version and haven't been deprecated beyond the version limits.
func (vv *VersionValidator) validateVariableVersions(program *ast.Program, vclVersion int) {
	// Visit all subroutines and validate variable version compatibility
	for _, decl := range program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok {
			vv.validateSubroutineVariableVersions(subDecl, vclVersion)
		}
	}
}

// validateSubroutineVariableVersions validates variable version compatibility in a subroutine
func (vv *VersionValidator) validateSubroutineVariableVersions(sub *ast.SubDecl, vclVersion int) {
	// Walk the AST and find variable accesses
	vv.walkStatementsForVersion(sub.Body.Statements, vclVersion)
}

// walkStatementsForVersion traverses statement AST nodes to identify variable references and
// validate their version compatibility. Recursively processes control structures and nested
// statements to ensure comprehensive version validation coverage.
func (vv *VersionValidator) walkStatementsForVersion(statements []ast.Statement, vclVersion int) {
	for _, stmt := range statements {
		switch s := stmt.(type) {
		case *ast.SetStatement:
			vv.validateVariableVersion(s.Variable, vclVersion)
		case *ast.UnsetStatement:
			vv.validateVariableVersion(s.Variable, vclVersion)
		case *ast.IfStatement:
			vv.validateExpressionVersion(s.Condition, vclVersion)
			if s.Then != nil {
				if block, ok := s.Then.(*ast.BlockStatement); ok {
					vv.walkStatementsForVersion(block.Statements, vclVersion)
				}
			}
			if s.Else != nil {
				if block, ok := s.Else.(*ast.BlockStatement); ok {
					vv.walkStatementsForVersion(block.Statements, vclVersion)
				}
			}
		case *ast.CallStatement:
			// Handle function call expressions that might contain variable references
			vv.validateExpressionVersion(s.Function, vclVersion)
		case *ast.ReturnStatement:
			if s.Action != nil {
				vv.validateExpressionVersion(s.Action, vclVersion)
			}
		case *ast.BlockStatement:
			vv.walkStatementsForVersion(s.Statements, vclVersion)
		}
	}
}

// validateExpressionVersion validates variable references in expressions
func (vv *VersionValidator) validateExpressionVersion(expr ast.Expression, vclVersion int) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *ast.MemberExpression:
		vv.validateVariableVersion(e, vclVersion)
	case *ast.Identifier:
		vv.validateVariableVersion(e, vclVersion)
	case *ast.BinaryExpression:
		vv.validateExpressionVersion(e.Left, vclVersion)
		vv.validateExpressionVersion(e.Right, vclVersion)
	case *ast.UnaryExpression:
		vv.validateExpressionVersion(e.Operand, vclVersion)
	case *ast.CallExpression:
		for _, arg := range e.Arguments {
			vv.validateExpressionVersion(arg, vclVersion)
		}
	}
}

// validateVariableVersion validates a specific variable against version constraints
func (vv *VersionValidator) validateVariableVersion(expr ast.Expression, vclVersion int) {
	varName := vv.extractVariableName(expr)
	if varName == "" {
		return
	}

	// Get variable metadata
	metadata, err := vv.loader.GetMetadata()
	if err != nil || metadata == nil {
		return
	}

	variable, exists := metadata.VCLVariables[varName]
	if !exists {
		// Check for dynamic variables like req.http.*, storage.*, etc.
		normalizedName := vv.normalizeDynamicVariableName(varName)
		if normalizedName != "" {
			if dynVar, dynExists := metadata.VCLVariables[normalizedName]; dynExists {
				variable = dynVar
			} else {
				return // Unknown variable, handled by other validators
			}
		} else {
			return // Unknown variable, handled by other validators
		}
	}

	// Check version compatibility
	if vclVersion < variable.VersionLow {
		vv.addError(fmt.Sprintf("variable '%s' requires VCL version %.1f or higher (current: %.1f)",
			varName, float64(variable.VersionLow)/10.0, float64(vclVersion)/10.0))
	}

	if vclVersion > variable.VersionHigh {
		vv.addError(fmt.Sprintf("variable '%s' is not available in VCL version %.1f (deprecated after %.1f)",
			varName, float64(vclVersion)/10.0, float64(variable.VersionHigh)/10.0))
	}
}

// extractVariableName extracts the variable name from an expression
func (vv *VersionValidator) extractVariableName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Name
	case *ast.MemberExpression:
		return vv.extractMemberVariableName(e)
	default:
		return ""
	}
}

// extractMemberVariableName extracts full variable name from member expression
func (vv *VersionValidator) extractMemberVariableName(expr *ast.MemberExpression) string {
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

// normalizeDynamicVariableName converts specific variable instances like 'req.http.host' or
// 'storage.memory.free_space' into their generic metadata patterns like 'req.http.' or 'storage.*'.
// Essential for validating dynamic VCL variables against their template definitions.
func (vv *VersionValidator) normalizeDynamicVariableName(varName string) string {
	// Handle req.http.*, bereq.http.*, beresp.http.*, resp.http.*, obj.http.*
	if strings.Contains(varName, ".http.") {
		parts := strings.Split(varName, ".http.")
		if len(parts) == 2 {
			return parts[0] + ".http."
		}
	}

	// Handle storage.* variables
	if strings.HasPrefix(varName, "storage.") {
		parts := strings.Split(varName, ".")
		if len(parts) >= 3 {
			// storage.<name>.property -> storage.<name>.*
			return "storage." + parts[1] + ".*"
		}
	}

	return ""
}

// addError adds an error message to the validator
func (vv *VersionValidator) addError(message string) {
	vv.errors = append(vv.errors, message)
}
