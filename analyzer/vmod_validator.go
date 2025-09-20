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
	ast.BaseVisitor
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
	ast.Accept(node, v)
	return v.errors
}

// VisitProgram implements ast.Visitor
func (v *VMODValidator) VisitProgram(program *ast.Program) interface{} {
	for _, decl := range program.Declarations {
		ast.Accept(decl, v)
	}
	return nil
}

// VisitSubDecl implements ast.Visitor
func (v *VMODValidator) VisitSubDecl(sub *ast.SubDecl) interface{} {
	// Set current method context for restriction validation
	oldMethod := v.currentMethod
	v.currentMethod = sub.Name
	defer func() { v.currentMethod = oldMethod }()

	for _, stmt := range sub.Body.Statements {
		ast.Accept(stmt, v)
	}
	return nil
}

// VisitImportDecl implements ast.Visitor
func (v *VMODValidator) VisitImportDecl(importDecl *ast.ImportDecl) interface{} {
	if err := v.registry.ValidateImport(importDecl.Module); err != nil {
		v.addError(fmt.Sprintf("import validation failed: %v", err))
		return nil
	}

	// Add module to symbol table
	if err := v.symbolTable.DefineModule(importDecl.Module); err != nil {
		v.addError(fmt.Sprintf("failed to register module %s: %v", importDecl.Module, err))
		return nil
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
	return nil
}

// VisitCallExpression implements ast.Visitor
func (v *VMODValidator) VisitCallExpression(callExpr *ast.CallExpression) interface{} {
	memberExpr, ok := callExpr.Function.(*ast.MemberExpression)
	if !ok {
		// Not a VMOD call, visit children normally
		ast.Accept(callExpr.Function, v)
		for _, arg := range callExpr.Arguments {
			ast.Accept(arg, v)
		}
		for _, arg := range callExpr.NamedArguments {
			ast.Accept(arg, v)
		}
		return nil
	}

	// Check if this is a VMOD function call or object method call
	if objIdent, ok := memberExpr.Object.(*ast.Identifier); ok {
		// Check if this identifier refers to a known VMOD object first
		objectSymbol := v.symbolTable.Lookup(objIdent.Name)
		if objectSymbol != nil && objectSymbol.Kind == types.SymbolVMODObject {
			// Object method call: object.method()
			v.validateObjectMethodCall(memberExpr, callExpr.Arguments, callExpr.NamedArguments)
		} else {
			// Treat as module function call: module.function()
			// This will handle both known and unknown modules appropriately
			v.validateModuleFunctionCall(memberExpr, callExpr.Arguments, callExpr.NamedArguments)
		}
	} else {
		// More complex expressions - treat as object method call
		v.validateObjectMethodCall(memberExpr, callExpr.Arguments, callExpr.NamedArguments)
	}

	// Visit positional arguments
	for _, arg := range callExpr.Arguments {
		ast.Accept(arg, v)
	}

	// Visit named arguments
	for _, arg := range callExpr.NamedArguments {
		ast.Accept(arg, v)
	}
	return nil
}

// VisitMemberExpression implements ast.Visitor
func (v *VMODValidator) VisitMemberExpression(memberExpr *ast.MemberExpression) interface{} {
	// Visit children
	ast.Accept(memberExpr.Object, v)
	ast.Accept(memberExpr.Property, v)
	return nil
}

// validateModuleFunctionCall validates a module function call
func (v *VMODValidator) validateModuleFunctionCall(memberExpr *ast.MemberExpression, args []ast.Expression, namedArgs map[string]ast.Expression) {
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

	// Get function definition to validate named arguments
	function, err := v.registry.GetFunction(moduleName, functionName)
	if err != nil {
		v.addError(fmt.Sprintf("VMOD function call validation failed: %v", err))
		return
	}

	// Build complete argument list combining positional and named arguments
	completeArgs, err := v.buildCompleteArgumentList(function, args, namedArgs)
	if err != nil {
		v.addError(fmt.Sprintf("Argument validation failed: %v", err))
		return
	}

	// Validate function call with enhanced type inference
	argTypes := v.extractArgumentTypesWithContext(moduleName, functionName, completeArgs)
	if err := v.registry.ValidateFunctionCall(moduleName, functionName, argTypes); err != nil {
		v.addError(fmt.Sprintf("VMOD function call validation failed: %v", err))
		return
	}

	// Validate function restrictions
	v.validateFunctionRestrictions(moduleName, functionName)
}

// validateObjectMethodCall validates an object method call
func (v *VMODValidator) validateObjectMethodCall(memberExpr *ast.MemberExpression, args []ast.Expression, namedArgs map[string]ast.Expression) {
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

	// TODO: Track the object's module and type by extending the Symbol struct to store more metadata
	// For this implementation, we'll assume the object is valid if it's in the symbol table
}

// fillPositionalArgs fills the result slice with positional arguments and marks them as used
func (v *VMODValidator) fillPositionalArgs(result []ast.Expression, parameterUsed []bool, function *vcc.Function, positionalArgs []ast.Expression) error {
	for i, arg := range positionalArgs {
		if i >= len(function.Parameters) {
			return fmt.Errorf("too many positional arguments: got %d, function accepts at most %d", len(positionalArgs), len(function.Parameters))
		}
		result[i] = arg
		parameterUsed[i] = true
	}
	return nil
}

// fillNamedArgs maps named arguments to their correct parameter positions
func (v *VMODValidator) fillNamedArgs(result []ast.Expression, parameterUsed []bool, function *vcc.Function, namedArgs map[string]ast.Expression) error {
	for argName, argValue := range namedArgs {
		// Find the parameter by name
		paramIndex := -1
		for i, param := range function.Parameters {
			if param.Name == argName {
				paramIndex = i
				break
			}
		}

		if paramIndex == -1 {
			return fmt.Errorf("unknown argument '%s'", argName)
		}

		if parameterUsed[paramIndex] {
			return fmt.Errorf("argument '%s' already provided as positional argument", argName)
		}

		result[paramIndex] = argValue
		parameterUsed[paramIndex] = true
	}
	return nil
}

// applyDefaultArgs checks for missing required parameters and handles defaults
func (v *VMODValidator) applyDefaultArgs(result []ast.Expression, parameterUsed []bool, function *vcc.Function) error {
	for i, param := range function.Parameters {
		if !parameterUsed[i] {
			if !param.Optional && param.DefaultValue == "" {
				return fmt.Errorf("missing required argument '%s'", param.Name)
			}
			// For optional parameters without provided values, we could insert default expressions
			// but for now we'll just leave them nil and let the existing validation handle it
		}
	}
	return nil
}

// buildCompleteArgumentList combines positional and named arguments into a complete argument list
func (v *VMODValidator) buildCompleteArgumentList(function *vcc.Function, positionalArgs []ast.Expression, namedArgs map[string]ast.Expression) ([]ast.Expression, error) {
	if function == nil {
		return positionalArgs, nil // Fallback if no function definition available
	}

	// Create a result slice with the same capacity as the function parameters
	result := make([]ast.Expression, len(function.Parameters))
	parameterUsed := make([]bool, len(function.Parameters))

	// Phase 1: Fill in positional arguments
	if err := v.fillPositionalArgs(result, parameterUsed, function, positionalArgs); err != nil {
		return nil, err
	}

	// Phase 2: Fill in named arguments
	if err := v.fillNamedArgs(result, parameterUsed, function, namedArgs); err != nil {
		return nil, err
	}

	// Phase 3: Check for missing required parameters and apply defaults
	if err := v.applyDefaultArgs(result, parameterUsed, function); err != nil {
		return nil, err
	}

	// Don't trim the result - we need to maintain positional mapping
	// The validation logic should handle nil arguments for optional parameters
	return result, nil
}

// validateNewStatement validates a VMOD object instantiation statement
// VisitNewStatement implements ast.Visitor
func (v *VMODValidator) VisitNewStatement(newStmt *ast.NewStatement) interface{} {
	// Extract variable name being assigned
	varName, ok := newStmt.Name.(*ast.Identifier)
	if !ok {
		v.addError("new statement: variable name must be an identifier")
		return nil
	}

	// Extract VMOD constructor call
	constructorCall, ok := newStmt.Constructor.(*ast.CallExpression)
	if !ok {
		v.addError("new statement: constructor must be a function call")
		return nil
	}

	// Extract module.object() call
	memberExpr, ok := constructorCall.Function.(*ast.MemberExpression)
	if !ok {
		v.addError("new statement: constructor must be a module.object() call")
		return nil
	}

	moduleIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok {
		v.addError("new statement: module name must be an identifier")
		return nil
	}

	objectIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		v.addError("new statement: object name must be an identifier")
		return nil
	}

	moduleName := moduleIdent.Name
	objectName := objectIdent.Name

	// Check if module is imported
	if !v.symbolTable.IsModuleImported(moduleName) {
		v.addError(fmt.Sprintf("module %s is not imported", moduleName))
		return nil
	}

	// Validate object construction with enhanced type inference
	argTypes := v.extractArgumentTypesWithObjectContext(moduleName, objectName, constructorCall.Arguments)
	if err := v.registry.ValidateObjectConstruction(moduleName, objectName, argTypes); err != nil {
		v.addError(fmt.Sprintf("VMOD object construction validation failed: %v", err))
		return nil
	}

	// Register the object instance in the symbol table
	if err := v.symbolTable.DefineVMODObject(varName.Name, moduleName, objectName); err != nil {
		v.addError(fmt.Sprintf("failed to register VMOD object %s: %v", varName.Name, err))
		return nil
	}

	// Visit constructor arguments for nested validation
	for _, arg := range constructorCall.Arguments {
		ast.Accept(arg, v)
	}
	return nil
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

// extractArgumentTypesWithParameters extracts VCC types from AST expressions using provided parameter definitions
func (v *VMODValidator) extractArgumentTypesWithParameters(args []ast.Expression, parameters []vcc.Parameter) []vcc.VCCType {
	var types []vcc.VCCType

	for i, arg := range args {
		var expectedType vcc.VCCType
		if i < len(parameters) {
			expectedType = parameters[i].Type
		}

		// Handle nil arguments (for optional parameters)
		if arg == nil {
			// For optional parameters, use the expected type
			types = append(types, expectedType)
		} else {
			vccType := v.inferExpressionType(arg, expectedType)
			types = append(types, vccType)
		}
	}

	return types
}

// extractArgumentTypesWithContext extracts VCC types from AST expressions with parameter context
func (v *VMODValidator) extractArgumentTypesWithContext(moduleName, functionName string, args []ast.Expression) []vcc.VCCType {
	// Look up function to get expected parameter types
	function, err := v.registry.GetFunction(moduleName, functionName)
	if err != nil {
		// Fallback to basic type inference if function not found
		return v.extractArgumentTypes(args)
	}

	return v.extractArgumentTypesWithParameters(args, function.Parameters)
}

// extractArgumentTypesWithObjectContext extracts VCC types from AST expressions with object constructor context
func (v *VMODValidator) extractArgumentTypesWithObjectContext(moduleName, objectName string, args []ast.Expression) []vcc.VCCType {
	// Look up object to get expected constructor parameter types
	object, err := v.registry.GetObject(moduleName, objectName)
	if err != nil {
		// Fallback to basic type inference if object not found
		return v.extractArgumentTypes(args)
	}

	return v.extractArgumentTypesWithParameters(args, object.Constructor)
}

// inferExpressionType infers the VCC type of an expression
// If expectedType is provided, it enables enhanced type inference with context
func (v *VMODValidator) inferExpressionType(expr ast.Expression, expectedType ...vcc.VCCType) vcc.VCCType {
	var expected vcc.VCCType
	if len(expectedType) > 0 {
		expected = expectedType[0]
	}

	switch e := expr.(type) {
	case *ast.StringLiteral:
		return vcc.TypeString
	case *ast.IntegerLiteral:
		// If we have expected type context, check if we can coerce INT to the expected type
		if expected != "" && v.isTypeCompatible(vcc.TypeInt, expected) {
			return expected
		}
		return vcc.TypeInt
	case *ast.FloatLiteral:
		return vcc.TypeReal
	case *ast.BooleanLiteral:
		return vcc.TypeBool
	case *ast.TimeExpression:
		return vcc.TypeDuration
	case *ast.Identifier:
		// Look up identifier in symbol table first
		symbol := v.symbolTable.Lookup(e.Name)
		if symbol != nil {
			return v.convertSymbolTypeToVCCType(symbol.Type)
		}
		// If expected type is BOOL and this is a boolean literal identifier, treat it as bool
		if expected == vcc.TypeBool && (e.Name == "true" || e.Name == "false") {
			return vcc.TypeBool
		}
		// If expected type is ENUM and this is a bare identifier, treat it as enum
		if expected == vcc.TypeEnum {
			return vcc.TypeEnum
		}
		return vcc.TypeString // Default assumption
	case *ast.MemberExpression:
		// Try to infer method return type for VMOD objects
		if returnType := v.inferObjectMethodReturnType(e); returnType != "" {
			return returnType
		}
		return vcc.TypeString // Default assumption
	case *ast.CallExpression:
		// For call expressions, try to look up the return type
		if returnType := v.inferCallExpressionReturnType(e); returnType != "" {
			return returnType
		}
		return vcc.TypeString // Default assumption
	case *ast.UnaryExpression:
		// For unary expressions, infer the type of the operand with context if available
		// This handles cases like "-1s" where the whole expression should be treated as the operand's type
		if expected != "" {
			return v.inferExpressionType(e.Operand, expected)
		}
		return v.inferExpressionType(e.Operand)
	default:
		return vcc.TypeString // Default assumption
	}
}

// inferCallExpressionReturnType attempts to infer the return type of a VMOD function call
func (v *VMODValidator) inferCallExpressionReturnType(callExpr *ast.CallExpression) vcc.VCCType {
	// Check if this is a VMOD function call (module.function())
	memberExpr, ok := callExpr.Function.(*ast.MemberExpression)
	if !ok {
		return "" // Not a member call
	}

	moduleIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok {
		return "" // Object is not an identifier
	}

	functionIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		return "" // Property is not an identifier
	}

	moduleOrObjectName := moduleIdent.Name
	functionOrMethodName := functionIdent.Name

	// First check if this is a module function call (module.function())
	if v.symbolTable.IsModuleImported(moduleOrObjectName) {
		// Look up function in registry
		function, err := v.registry.GetFunction(moduleOrObjectName, functionOrMethodName)
		if err != nil {
			return "" // Function not found
		}
		return function.ReturnType
	}

	// Check if this is an object method call (object.method())
	objectSymbol := v.symbolTable.Lookup(moduleOrObjectName)
	if objectSymbol != nil && objectSymbol.Kind == types.SymbolVMODObject {
		// Check that object has required metadata
		if objectSymbol.ModuleName == "" || objectSymbol.ObjectType == "" {
			return "" // Object missing required VMOD metadata
		}

		// Look up method in registry using object metadata
		method, err := v.registry.GetMethod(objectSymbol.ModuleName, objectSymbol.ObjectType, functionOrMethodName)
		if err != nil {
			return "" // Method not found
		}
		return method.ReturnType
	}

	return "" // Not a recognized module function or object method call
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
	case types.HTTP:
		return vcc.TypeHTTP
	default:
		return vcc.TypeString // Default
	}
}

// isTypeCompatible checks if a given type can be coerced to the expected type
func (v *VMODValidator) isTypeCompatible(got, expected vcc.VCCType) bool {
	// Exact match
	if got == expected {
		return true
	}

	// Allow INT to REAL coercion (common in VCL)
	if got == vcc.TypeInt && expected == vcc.TypeReal {
		return true
	}

	// TODO: Add other type coercions as needed
	return false
}

// inferObjectMethodReturnType attempts to infer the return type of a VMOD object method call
func (v *VMODValidator) inferObjectMethodReturnType(memberExpr *ast.MemberExpression) vcc.VCCType {
	// Check if this is an object.method() pattern
	objectIdent, ok := memberExpr.Object.(*ast.Identifier)
	if !ok {
		return "" // Object is not an identifier
	}

	methodIdent, ok := memberExpr.Property.(*ast.Identifier)
	if !ok {
		return "" // Property is not an identifier
	}

	objectName := objectIdent.Name
	methodName := methodIdent.Name

	// Look up object in symbol table
	objectSymbol := v.symbolTable.Lookup(objectName)
	if objectSymbol == nil || objectSymbol.Kind != types.SymbolVMODObject {
		return "" // Object not found or not a VMOD object
	}

	// Check that object has required metadata
	if objectSymbol.ModuleName == "" || objectSymbol.ObjectType == "" {
		return "" // Object missing required VMOD metadata
	}

	// Look up method in registry using object metadata
	method, err := v.registry.GetMethod(objectSymbol.ModuleName, objectSymbol.ObjectType, methodName)
	if err != nil {
		// Method not found in registry, return empty type
		return ""
	}

	return method.ReturnType
}

// VisitBlockStatement implements ast.Visitor
func (v *VMODValidator) VisitBlockStatement(node *ast.BlockStatement) interface{} {
	for _, stmt := range node.Statements {
		ast.Accept(stmt, v)
	}
	return nil
}

// VisitExpressionStatement implements ast.Visitor
func (v *VMODValidator) VisitExpressionStatement(node *ast.ExpressionStatement) interface{} {
	ast.Accept(node.Expression, v)
	return nil
}

// VisitIfStatement implements ast.Visitor
func (v *VMODValidator) VisitIfStatement(node *ast.IfStatement) interface{} {
	ast.Accept(node.Condition, v)
	ast.Accept(node.Then, v)
	if node.Else != nil {
		ast.Accept(node.Else, v)
	}
	return nil
}

// VisitSetStatement implements ast.Visitor
func (v *VMODValidator) VisitSetStatement(node *ast.SetStatement) interface{} {
	ast.Accept(node.Variable, v)
	ast.Accept(node.Value, v)
	return nil
}

// VisitUnsetStatement implements ast.Visitor
func (v *VMODValidator) VisitUnsetStatement(node *ast.UnsetStatement) interface{} {
	ast.Accept(node.Variable, v)
	return nil
}

// VisitReturnStatement implements ast.Visitor
func (v *VMODValidator) VisitReturnStatement(node *ast.ReturnStatement) interface{} {
	if node.Action != nil {
		ast.Accept(node.Action, v)
	}
	return nil
}

// VisitCallStatement implements ast.Visitor
func (v *VMODValidator) VisitCallStatement(node *ast.CallStatement) interface{} {
	ast.Accept(node.Function, v)
	return nil
}

// VisitBinaryExpression implements ast.Visitor
func (v *VMODValidator) VisitBinaryExpression(node *ast.BinaryExpression) interface{} {
	ast.Accept(node.Left, v)
	ast.Accept(node.Right, v)
	return nil
}

// VisitUnaryExpression implements ast.Visitor
func (v *VMODValidator) VisitUnaryExpression(node *ast.UnaryExpression) interface{} {
	ast.Accept(node.Operand, v)
	return nil
}

// addError adds a validation error
func (v *VMODValidator) addError(message string) {
	v.errors = append(v.errors, message)
}

// Errors returns all validation errors
func (v *VMODValidator) Errors() []string {
	return v.errors
}
