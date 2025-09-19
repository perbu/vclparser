package ast

// Visitor provides an interface for traversing AST nodes
type Visitor interface {
	VisitProgram(*Program) interface{}
	VisitVCLVersionDecl(*VCLVersionDecl) interface{}
	VisitImportDecl(*ImportDecl) interface{}
	VisitIncludeDecl(*IncludeDecl) interface{}
	VisitBackendDecl(*BackendDecl) interface{}
	VisitProbeDecl(*ProbeDecl) interface{}
	VisitACLDecl(*ACLDecl) interface{}
	VisitSubDecl(*SubDecl) interface{}

	VisitBlockStatement(*BlockStatement) interface{}
	VisitExpressionStatement(*ExpressionStatement) interface{}
	VisitIfStatement(*IfStatement) interface{}
	VisitSetStatement(*SetStatement) interface{}
	VisitUnsetStatement(*UnsetStatement) interface{}
	VisitCallStatement(*CallStatement) interface{}
	VisitReturnStatement(*ReturnStatement) interface{}
	VisitSyntheticStatement(*SyntheticStatement) interface{}
	VisitErrorStatement(*ErrorStatement) interface{}
	VisitRestartStatement(*RestartStatement) interface{}
	VisitCSourceStatement(*CSourceStatement) interface{}

	VisitBinaryExpression(*BinaryExpression) interface{}
	VisitUnaryExpression(*UnaryExpression) interface{}
	VisitCallExpression(*CallExpression) interface{}
	VisitMemberExpression(*MemberExpression) interface{}
	VisitIndexExpression(*IndexExpression) interface{}
	VisitParenthesizedExpression(*ParenthesizedExpression) interface{}
	VisitConditionalExpression(*ConditionalExpression) interface{}
	VisitRegexMatchExpression(*RegexMatchExpression) interface{}
	VisitAssignmentExpression(*AssignmentExpression) interface{}
	VisitUpdateExpression(*UpdateExpression) interface{}
	VisitArrayExpression(*ArrayExpression) interface{}
	VisitObjectExpression(*ObjectExpression) interface{}
	VisitVariableExpression(*VariableExpression) interface{}
	VisitTimeExpression(*TimeExpression) interface{}
	VisitIPExpression(*IPExpression) interface{}

	VisitIdentifier(*Identifier) interface{}
	VisitStringLiteral(*StringLiteral) interface{}
	VisitIntegerLiteral(*IntegerLiteral) interface{}
	VisitFloatLiteral(*FloatLiteral) interface{}
	VisitBooleanLiteral(*BooleanLiteral) interface{}
	VisitDurationLiteral(*DurationLiteral) interface{}
}

// Accept calls the appropriate visit method on the visitor
func Accept(node Node, visitor Visitor) interface{} {
	switch n := node.(type) {
	case *Program:
		return visitor.VisitProgram(n)
	case *VCLVersionDecl:
		return visitor.VisitVCLVersionDecl(n)
	case *ImportDecl:
		return visitor.VisitImportDecl(n)
	case *IncludeDecl:
		return visitor.VisitIncludeDecl(n)
	case *BackendDecl:
		return visitor.VisitBackendDecl(n)
	case *ProbeDecl:
		return visitor.VisitProbeDecl(n)
	case *ACLDecl:
		return visitor.VisitACLDecl(n)
	case *SubDecl:
		return visitor.VisitSubDecl(n)

	case *BlockStatement:
		return visitor.VisitBlockStatement(n)
	case *ExpressionStatement:
		return visitor.VisitExpressionStatement(n)
	case *IfStatement:
		return visitor.VisitIfStatement(n)
	case *SetStatement:
		return visitor.VisitSetStatement(n)
	case *UnsetStatement:
		return visitor.VisitUnsetStatement(n)
	case *CallStatement:
		return visitor.VisitCallStatement(n)
	case *ReturnStatement:
		return visitor.VisitReturnStatement(n)
	case *SyntheticStatement:
		return visitor.VisitSyntheticStatement(n)
	case *ErrorStatement:
		return visitor.VisitErrorStatement(n)
	case *RestartStatement:
		return visitor.VisitRestartStatement(n)
	case *CSourceStatement:
		return visitor.VisitCSourceStatement(n)

	case *BinaryExpression:
		return visitor.VisitBinaryExpression(n)
	case *UnaryExpression:
		return visitor.VisitUnaryExpression(n)
	case *CallExpression:
		return visitor.VisitCallExpression(n)
	case *MemberExpression:
		return visitor.VisitMemberExpression(n)
	case *IndexExpression:
		return visitor.VisitIndexExpression(n)
	case *ParenthesizedExpression:
		return visitor.VisitParenthesizedExpression(n)
	case *ConditionalExpression:
		return visitor.VisitConditionalExpression(n)
	case *RegexMatchExpression:
		return visitor.VisitRegexMatchExpression(n)
	case *AssignmentExpression:
		return visitor.VisitAssignmentExpression(n)
	case *UpdateExpression:
		return visitor.VisitUpdateExpression(n)
	case *ArrayExpression:
		return visitor.VisitArrayExpression(n)
	case *ObjectExpression:
		return visitor.VisitObjectExpression(n)
	case *VariableExpression:
		return visitor.VisitVariableExpression(n)
	case *TimeExpression:
		return visitor.VisitTimeExpression(n)
	case *IPExpression:
		return visitor.VisitIPExpression(n)

	case *Identifier:
		return visitor.VisitIdentifier(n)
	case *StringLiteral:
		return visitor.VisitStringLiteral(n)
	case *IntegerLiteral:
		return visitor.VisitIntegerLiteral(n)
	case *FloatLiteral:
		return visitor.VisitFloatLiteral(n)
	case *BooleanLiteral:
		return visitor.VisitBooleanLiteral(n)
	case *DurationLiteral:
		return visitor.VisitDurationLiteral(n)

	default:
		panic("unknown node type")
	}
}

// BaseVisitor provides a default implementation of the Visitor interface
// Embed this in custom visitors and override only the methods you need
type BaseVisitor struct{}

func (bv *BaseVisitor) VisitProgram(node *Program) interface{}                         { return nil }
func (bv *BaseVisitor) VisitVCLVersionDecl(node *VCLVersionDecl) interface{}           { return nil }
func (bv *BaseVisitor) VisitImportDecl(node *ImportDecl) interface{}                   { return nil }
func (bv *BaseVisitor) VisitIncludeDecl(node *IncludeDecl) interface{}                 { return nil }
func (bv *BaseVisitor) VisitBackendDecl(node *BackendDecl) interface{}                 { return nil }
func (bv *BaseVisitor) VisitProbeDecl(node *ProbeDecl) interface{}                     { return nil }
func (bv *BaseVisitor) VisitACLDecl(node *ACLDecl) interface{}                         { return nil }
func (bv *BaseVisitor) VisitSubDecl(node *SubDecl) interface{}                         { return nil }
func (bv *BaseVisitor) VisitBlockStatement(node *BlockStatement) interface{}           { return nil }
func (bv *BaseVisitor) VisitExpressionStatement(node *ExpressionStatement) interface{} { return nil }
func (bv *BaseVisitor) VisitIfStatement(node *IfStatement) interface{}                 { return nil }
func (bv *BaseVisitor) VisitSetStatement(node *SetStatement) interface{}               { return nil }
func (bv *BaseVisitor) VisitUnsetStatement(node *UnsetStatement) interface{}           { return nil }
func (bv *BaseVisitor) VisitCallStatement(node *CallStatement) interface{}             { return nil }
func (bv *BaseVisitor) VisitReturnStatement(node *ReturnStatement) interface{}         { return nil }
func (bv *BaseVisitor) VisitSyntheticStatement(node *SyntheticStatement) interface{}   { return nil }
func (bv *BaseVisitor) VisitErrorStatement(node *ErrorStatement) interface{}           { return nil }
func (bv *BaseVisitor) VisitRestartStatement(node *RestartStatement) interface{}       { return nil }
func (bv *BaseVisitor) VisitCSourceStatement(node *CSourceStatement) interface{}       { return nil }
func (bv *BaseVisitor) VisitBinaryExpression(node *BinaryExpression) interface{}       { return nil }
func (bv *BaseVisitor) VisitUnaryExpression(node *UnaryExpression) interface{}         { return nil }
func (bv *BaseVisitor) VisitCallExpression(node *CallExpression) interface{}           { return nil }
func (bv *BaseVisitor) VisitMemberExpression(node *MemberExpression) interface{}       { return nil }
func (bv *BaseVisitor) VisitIndexExpression(node *IndexExpression) interface{}         { return nil }
func (bv *BaseVisitor) VisitParenthesizedExpression(node *ParenthesizedExpression) interface{} {
	return nil
}
func (bv *BaseVisitor) VisitConditionalExpression(node *ConditionalExpression) interface{} {
	return nil
}
func (bv *BaseVisitor) VisitRegexMatchExpression(node *RegexMatchExpression) interface{} {
	return nil
}
func (bv *BaseVisitor) VisitAssignmentExpression(node *AssignmentExpression) interface{} {
	return nil
}
func (bv *BaseVisitor) VisitUpdateExpression(node *UpdateExpression) interface{}     { return nil }
func (bv *BaseVisitor) VisitArrayExpression(node *ArrayExpression) interface{}       { return nil }
func (bv *BaseVisitor) VisitObjectExpression(node *ObjectExpression) interface{}     { return nil }
func (bv *BaseVisitor) VisitVariableExpression(node *VariableExpression) interface{} { return nil }
func (bv *BaseVisitor) VisitTimeExpression(node *TimeExpression) interface{}         { return nil }
func (bv *BaseVisitor) VisitIPExpression(node *IPExpression) interface{}             { return nil }
func (bv *BaseVisitor) VisitIdentifier(node *Identifier) interface{}                 { return nil }
func (bv *BaseVisitor) VisitStringLiteral(node *StringLiteral) interface{}           { return nil }
func (bv *BaseVisitor) VisitIntegerLiteral(node *IntegerLiteral) interface{}         { return nil }
func (bv *BaseVisitor) VisitFloatLiteral(node *FloatLiteral) interface{}             { return nil }
func (bv *BaseVisitor) VisitBooleanLiteral(node *BooleanLiteral) interface{}         { return nil }
func (bv *BaseVisitor) VisitDurationLiteral(node *DurationLiteral) interface{}       { return nil }
