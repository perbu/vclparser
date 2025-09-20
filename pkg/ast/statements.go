package ast

// BlockStatement represents a block of statements ({ ... })
type BlockStatement struct {
	BaseNode
	Statements []Statement
}

func (bs *BlockStatement) String() string { return "BlockStatement" }
func (bs *BlockStatement) statementNode() {}

// ExpressionStatement represents an expression used as a statement
type ExpressionStatement struct {
	BaseNode
	Expression Expression
}

func (es *ExpressionStatement) String() string { return "ExpressionStatement" }
func (es *ExpressionStatement) statementNode() {}

// IfStatement represents an if/else statement
type IfStatement struct {
	BaseNode
	Condition Expression
	Then      Statement
	Else      Statement // optional
}

func (is *IfStatement) String() string { return "IfStatement" }
func (is *IfStatement) statementNode() {}

// SetStatement represents a set statement (variable assignment)
type SetStatement struct {
	BaseNode
	Variable Expression
	Operator string // "=", "+=", "-=", "*=", "/="
	Value    Expression
}

func (ss *SetStatement) String() string { return "SetStatement" }
func (ss *SetStatement) statementNode() {}

// UnsetStatement represents an unset statement
type UnsetStatement struct {
	BaseNode
	Variable Expression
}

func (us *UnsetStatement) String() string { return "UnsetStatement" }
func (us *UnsetStatement) statementNode() {}

// CallStatement represents a function call statement
type CallStatement struct {
	BaseNode
	Function Expression
}

func (cs *CallStatement) String() string { return "CallStatement" }
func (cs *CallStatement) statementNode() {}

// ReturnStatement represents a return statement
type ReturnStatement struct {
	BaseNode
	Action Expression // The return action (hash, pass, etc.)
}

func (rs *ReturnStatement) String() string { return "ReturnStatement" }
func (rs *ReturnStatement) statementNode() {}

// SyntheticStatement represents a synthetic statement
type SyntheticStatement struct {
	BaseNode
	Response Expression
}

func (ss *SyntheticStatement) String() string { return "SyntheticStatement" }
func (ss *SyntheticStatement) statementNode() {}

// ErrorStatement represents an error statement
type ErrorStatement struct {
	BaseNode
	Code     Expression // optional
	Response Expression // optional
}

func (es *ErrorStatement) String() string { return "ErrorStatement" }
func (es *ErrorStatement) statementNode() {}

// RestartStatement represents a restart statement
type RestartStatement struct {
	BaseNode
}

func (rs *RestartStatement) String() string { return "RestartStatement" }
func (rs *RestartStatement) statementNode() {}

// CSourceStatement represents inline C code
type CSourceStatement struct {
	BaseNode
	Code string
}

func (cs *CSourceStatement) String() string { return "CSourceStatement" }
func (cs *CSourceStatement) statementNode() {}

// NewStatement represents a VMOD object instantiation statement
type NewStatement struct {
	BaseNode
	Name        Expression // The variable name being assigned to
	Constructor Expression // The VMOD constructor call expression
}

func (ns *NewStatement) String() string { return "NewStatement" }
func (ns *NewStatement) statementNode() {}
