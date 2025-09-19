package analyzer

import (
	"fmt"
	"strings"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/types"
	"github.com/varnish/vclparser/vcc"
	"github.com/varnish/vclparser/vmod"
)

// VMODValidator validates VMOD usage in VCL code
type VMODValidator struct {
	registry      *vmod.Registry
	symbolTable   *types.SymbolTable
	errors        []string
	currentMethod string // Current VCL method context
}

// NewVMODValidator creates a new VMOD validator
func NewVMODValidator(registry *vmod.Registry, symbolTable *types.SymbolTable) *VMODValidator {
	return &VMODValidator{
		registry:    registry,
		symbolTable: symbolTable,
		errors:      []string{},
	}
}

// Validate validates VMOD usage in an AST node
func (v *VMODValidator) Validate(node ast.Node) []string {
	v.errors = []string{}
	v.visit(node)
	return v.errors
}

// visit recursively visits AST nodes
func (v *VMODValidator) visit(node ast.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *ast.Program:
		v.visitProgram(n)
	case *ast.ImportDecl:
		v.validateImport(n)
	case *ast.CallExpression:
		v.validateCallExpression(n)
	case *ast.MemberExpression:
		v.validateMemberExpression(n)
	case *ast.SubDecl:
		v.visitSubroutine(n)
	case *ast.NewStatement:
		v.validateNewStatement(n)
	default:
		// Visit children for other node types
		v.visitChildren(node)
	}
}

// visitProgram validates a program node
func (v *VMODValidator) visitProgram(program *ast.Program) {
	for _, decl := range program.Declarations {
		v.visit(decl)
	}
}

// visitSubroutine validates a subroutine
func (v *VMODValidator) visitSubroutine(sub *ast.SubDecl) {
	// Set current method context for restriction validation
	oldMethod := v.currentMethod
	v.currentMethod = sub.Name
	defer func() { v.currentMethod = oldMethod }()

	for _, stmt := range sub.Body.Statements {
		v.visit(stmt)
	}
}

// validateImport validates an import declaration
func (v *VMODValidator) validateImport(importDecl *ast.ImportDecl) {
	if err := v.registry.ValidateImport(importDecl.Module); err != nil {
		v.addError(fmt.Sprintf("import validation failed: %v", err))
		return
	}

	// Add module to symbol table
	if err := v.symbolTable.DefineModule(importDecl.Module); err != nil {
		v.addError(fmt.Sprintf("failed to register module %s: %v", importDecl.Module, err))
		return
	}

	// Add VMOD functions to symbol table
	module, exists := v.registry.GetModule(importDecl.Module)
	if exists {
		for _, function := range module.Functions {
			returnType := v.convertVCCTypeToSymbolType(function.ReturnType)
			if err := v.symbolTable.DefineVMODFunction(importDecl.Module, function.Name, returnType); err != nil {
				v.addError(fmt.Sprintf("failed to register VMOD function %s.%s: %v",
					importDecl.Module, function.Name, err))
			}
		}
	}
}

// validateCallExpression validates function calls
func (v *VMODValidator) validateCallExpression(callExpr *ast.CallExpression) {
	memberExpr, ok := callExpr.Function.(*ast.MemberExpression)
	if !ok {
		// Not a VMOD call, skip
		v.visitChildren(callExpr)
		return
	}

	// Check if this is a VMOD function call or object method call
	if objIdent, ok := memberExpr.Object.(*ast.Identifier); ok {
		// Check if this identifier refers to a known VMOD object first
		objectSymbol := v.symbolTable.Lookup(objIdent.Name)
		if objectSymbol != nil && objectSymbol.Kind == types.SymbolVMODObject {
			// Object method call: object.method()
			v.validateObjectMethodCall(memberExpr, callExpr.Arguments)
		} else {
			// Treat as module function call: module.function()
			// This will handle both known and unknown modules appropriately
			v.validateModuleFunctionCall(memberExpr, callExpr.Arguments)
		}
	} else {
		// More complex expressions - treat as object method call
		v.validateObjectMethodCall(memberExpr, callExpr.Arguments)
	}

	// Visit arguments
	for _, arg := range callExpr.Arguments {
		v.visit(arg)
	}
}

// validateMemberExpression validates member access
func (v *VMODValidator) validateMemberExpression(memberExpr *ast.MemberExpression) {
	// Visit children
	v.visit(memberExpr.Object)
	v.visit(memberExpr.Property)
}

// validateModuleFunctionCall validates a module function call
func (v *VMODValidator) validateModuleFunctionCall(memberExpr *ast.MemberExpression, args []ast.Expression) {
	moduleIdent := memberExpr.Object.(*ast.Identifier)
	functionIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		v.addError("function name must be an identifier")
		return
	}

	moduleName := moduleIdent.Name
	functionName := functionIdent.Name

	// Check if module is imported
	if !v.symbolTable.IsModuleImported(moduleName) {
		v.addError(fmt.Sprintf("module %s is not imported", moduleName))
		return
	}

	// Validate function call
	argTypes := v.extractArgumentTypes(args)
	if err := v.registry.ValidateFunctionCall(moduleName, functionName, argTypes); err != nil {
		v.addError(fmt.Sprintf("VMOD function call validation failed: %v", err))
		return
	}

	// Validate function restrictions
	v.validateFunctionRestrictions(moduleName, functionName)
}

// validateObjectMethodCall validates an object method call
func (v *VMODValidator) validateObjectMethodCall(memberExpr *ast.MemberExpression, args []ast.Expression) {
	objectIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok {
		v.addError("object name must be an identifier")
		return
	}

	methodIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		v.addError("method name must be an identifier")
		return
	}

	objectName := objectIdent.Name
	_ = methodIdent.Name // methodName - not used in current implementation

	// Look up object in symbol table
	objectSymbol := v.symbolTable.Lookup(objectName)
	if objectSymbol == nil {
		v.addError(fmt.Sprintf("object %s is not defined", objectName))
		return
	}

	if objectSymbol.Kind != types.SymbolVMODObject {
		// Not a VMOD object, skip validation
		return
	}

	// For now, we'll need to track the object's module and type
	// This would require extending the Symbol struct to store more metadata
	// For this implementation, we'll assume the object is valid if it's in the symbol table
}

// validateNewStatement validates a VMOD object instantiation statement
func (v *VMODValidator) validateNewStatement(newStmt *ast.NewStatement) {
	// Extract variable name being assigned
	varName, ok := newStmt.Name.(*ast.Identifier)
	if !ok {
		v.addError("new statement: variable name must be an identifier")
		return
	}

	// Extract VMOD constructor call
	constructorCall, ok := newStmt.Constructor.(*ast.CallExpression)
	if !ok {
		v.addError("new statement: constructor must be a function call")
		return
	}

	// Extract module.object() call
	memberExpr, ok := constructorCall.Function.(*ast.MemberExpression)
	if !ok {
		v.addError("new statement: constructor must be a module.object() call")
		return
	}

	moduleIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok {
		v.addError("new statement: module name must be an identifier")
		return
	}

	objectIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		v.addError("new statement: object name must be an identifier")
		return
	}

	moduleName := moduleIdent.Name
	objectName := objectIdent.Name

	// Check if module is imported
	if !v.symbolTable.IsModuleImported(moduleName) {
		v.addError(fmt.Sprintf("module %s is not imported", moduleName))
		return
	}

	// Validate object construction
	argTypes := v.extractArgumentTypes(constructorCall.Arguments)
	if err := v.registry.ValidateObjectConstruction(moduleName, objectName, argTypes); err != nil {
		v.addError(fmt.Sprintf("VMOD object construction validation failed: %v", err))
		return
	}

	// Register the object instance in the symbol table
	if err := v.symbolTable.DefineVMODObject(varName.Name, moduleName, objectName); err != nil {
		v.addError(fmt.Sprintf("failed to register VMOD object %s: %v", varName.Name, err))
		return
	}

	// Visit constructor arguments for nested validation
	for _, arg := range constructorCall.Arguments {
		v.visit(arg)
	}
}

// validateFunctionRestrictions validates function usage restrictions
func (v *VMODValidator) validateFunctionRestrictions(moduleName, functionName string) {
	function, err := v.registry.GetFunction(moduleName, functionName)
	if err != nil {
		return // Error already reported
	}

	if len(function.Restrictions) == 0 {
		return // No restrictions
	}

	// Check if current method is allowed
	if v.currentMethod != "" {
		for _, allowedMethod := range function.Restrictions {
			if strings.EqualFold(allowedMethod, v.currentMethod) {
				return // Method is allowed
			}
		}
		v.addError(fmt.Sprintf("function %s.%s cannot be used in %s context",
			moduleName, functionName, v.currentMethod))
	}
}

// extractArgumentTypes extracts VCC types from AST expressions
func (v *VMODValidator) extractArgumentTypes(args []ast.Expression) []vcc.VCCType {
	var types []vcc.VCCType

	for _, arg := range args {
		vccType := v.inferExpressionType(arg)
		types = append(types, vccType)
	}

	return types
}

// inferExpressionType infers the VCC type of an expression
func (v *VMODValidator) inferExpressionType(expr ast.Expression) vcc.VCCType {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		return vcc.TypeString
	case *ast.IntegerLiteral:
		return vcc.TypeInt
	case *ast.FloatLiteral:
		return vcc.TypeReal
	case *ast.BooleanLiteral:
		return vcc.TypeBool
	case *ast.Identifier:
		// Look up identifier in symbol table
		symbol := v.symbolTable.Lookup(e.Name)
		if symbol != nil {
			return v.convertSymbolTypeToVCCType(symbol.Type)
		}
		return vcc.TypeString // Default assumption
	case *ast.MemberExpression:
		// For member expressions, try to infer the type
		return vcc.TypeString // Default assumption
	case *ast.CallExpression:
		// For call expressions, we'd need to look up the return type
		return vcc.TypeString // Default assumption
	default:
		return vcc.TypeString // Default assumption
	}
}

// convertVCCTypeToSymbolType converts VCC type to symbol table type
func (v *VMODValidator) convertVCCTypeToSymbolType(vccType vcc.VCCType) types.Type {
	switch vccType {
	case vcc.TypeString, vcc.TypeStringList, vcc.TypeStrands:
		return types.String
	case vcc.TypeInt:
		return types.Int
	case vcc.TypeReal:
		return types.Real
	case vcc.TypeBool:
		return types.Bool
	case vcc.TypeBackend:
		return types.Backend
	case vcc.TypeHeader:
		return types.Header
	case vcc.TypeDuration:
		return types.Duration
	case vcc.TypeBytes:
		return types.Bytes
	case vcc.TypeIP:
		return types.IP
	case vcc.TypeTime:
		return types.Time
	case vcc.TypeVoid:
		return types.Void
	default:
		return types.String // Default
	}
}

// convertSymbolTypeToVCCType converts symbol table type to VCC type
func (v *VMODValidator) convertSymbolTypeToVCCType(symbolType types.Type) vcc.VCCType {
	switch symbolType {
	case types.String:
		return vcc.TypeString
	case types.Int:
		return vcc.TypeInt
	case types.Real:
		return vcc.TypeReal
	case types.Bool:
		return vcc.TypeBool
	case types.Backend:
		return vcc.TypeBackend
	case types.Header:
		return vcc.TypeHeader
	case types.Duration:
		return vcc.TypeDuration
	case types.Bytes:
		return vcc.TypeBytes
	case types.IP:
		return vcc.TypeIP
	case types.Time:
		return vcc.TypeTime
	case types.Void:
		return vcc.TypeVoid
	default:
		return vcc.TypeString // Default
	}
}

// visitChildren visits all children of a node
func (v *VMODValidator) visitChildren(node ast.Node) {
	switch n := node.(type) {
	case *ast.Program:
		for _, decl := range n.Declarations {
			v.visit(decl)
		}
	case *ast.BlockStatement:
		for _, stmt := range n.Statements {
			v.visit(stmt)
		}
	case *ast.IfStatement:
		v.visit(n.Condition)
		v.visit(n.Then)
		if n.Else != nil {
			v.visit(n.Else)
		}
	case *ast.SetStatement:
		v.visit(n.Variable)
		v.visit(n.Value)
	case *ast.UnsetStatement:
		v.visit(n.Variable)
	case *ast.ReturnStatement:
		if n.Action != nil {
			v.visit(n.Action)
		}
	case *ast.CallStatement:
		v.visit(n.Function)
	case *ast.BinaryExpression:
		v.visit(n.Left)
		v.visit(n.Right)
	case *ast.UnaryExpression:
		v.visit(n.Operand)
	case *ast.CallExpression:
		v.visit(n.Function)
		for _, arg := range n.Arguments {
			v.visit(arg)
		}
	case *ast.MemberExpression:
		v.visit(n.Object)
		v.visit(n.Property)
	case *ast.NewStatement:
		v.visit(n.Name)
		v.visit(n.Constructor)
	}
}

// addError adds a validation error
func (v *VMODValidator) addError(message string) {
	v.errors = append(v.errors, message)
}

// Errors returns all validation errors
func (v *VMODValidator) Errors() []string {
	return v.errors
}
