package ast

// BinaryExpression represents a binary expression (e.g., a + b, a == b)
type BinaryExpression struct {
	BaseNode
	Left     Expression
	Operator string
	Right    Expression
}

func (be *BinaryExpression) String() string  { return "BinaryExpression(" + be.Operator + ")" }
func (be *BinaryExpression) expressionNode() {}

// UnaryExpression represents a unary expression (e.g., !a, -b)
type UnaryExpression struct {
	BaseNode
	Operator string
	Operand  Expression
}

func (ue *UnaryExpression) String() string  { return "UnaryExpression(" + ue.Operator + ")" }
func (ue *UnaryExpression) expressionNode() {}

// NamedArgument represents a named argument in a function call
type NamedArgument struct {
	Name  string
	Value Expression
}

// CallExpression represents a function call
type CallExpression struct {
	BaseNode
	Function       Expression
	Arguments      []Expression          // Positional arguments
	NamedArguments map[string]Expression // Named arguments: parameter_name -> value
}

func (ce *CallExpression) String() string  { return "CallExpression" }
func (ce *CallExpression) expressionNode() {}

// MemberExpression represents member access (e.g., req.url, obj.status)
type MemberExpression struct {
	BaseNode
	Object   Expression
	Property Expression
}

func (me *MemberExpression) String() string  { return "MemberExpression" }
func (me *MemberExpression) expressionNode() {}

// IndexExpression represents array/map indexing (e.g., headers["Host"])
type IndexExpression struct {
	BaseNode
	Object Expression
	Index  Expression
}

func (ie *IndexExpression) String() string  { return "IndexExpression" }
func (ie *IndexExpression) expressionNode() {}

// ParenthesizedExpression represents a parenthesized expression
type ParenthesizedExpression struct {
	BaseNode
	Expression Expression
}

func (pe *ParenthesizedExpression) String() string  { return "ParenthesizedExpression" }
func (pe *ParenthesizedExpression) expressionNode() {}

// RegexMatchExpression represents regex matching (~, !~)
type RegexMatchExpression struct {
	BaseNode
	Left     Expression
	Operator string // "~" or "!~"
	Right    Expression
}

func (re *RegexMatchExpression) String() string  { return "RegexMatchExpression(" + re.Operator + ")" }
func (re *RegexMatchExpression) expressionNode() {}

// AssignmentExpression represents assignment operations
type AssignmentExpression struct {
	BaseNode
	Left     Expression
	Operator string // "=", "+=", "-=", "*=", "/="
	Right    Expression
}

func (ae *AssignmentExpression) String() string  { return "AssignmentExpression(" + ae.Operator + ")" }
func (ae *AssignmentExpression) expressionNode() {}

// UpdateExpression represents increment/decrement operations (++, --)
type UpdateExpression struct {
	BaseNode
	Operator string // "++" or "--"
	Operand  Expression
	Prefix   bool // true for ++x, false for x++
}

func (ue *UpdateExpression) String() string  { return "UpdateExpression(" + ue.Operator + ")" }
func (ue *UpdateExpression) expressionNode() {}

// ArrayExpression represents array literals
type ArrayExpression struct {
	BaseNode
	Elements []Expression
}

func (ae *ArrayExpression) String() string  { return "ArrayExpression" }
func (ae *ArrayExpression) expressionNode() {}

// ObjectExpression represents object literals (key-value pairs)
type ObjectExpression struct {
	BaseNode
	Properties []*Property
}

func (oe *ObjectExpression) String() string  { return "ObjectExpression" }
func (oe *ObjectExpression) expressionNode() {}

// Property represents a property in an object expression
type Property struct {
	BaseNode
	Key   Expression
	Value Expression
}

func (p *Property) String() string { return "Property" }

// VariableExpression represents a variable reference
type VariableExpression struct {
	BaseNode
	Name string
}

func (ve *VariableExpression) String() string  { return "VariableExpression(" + ve.Name + ")" }
func (ve *VariableExpression) expressionNode() {}

// TimeExpression represents time literals with units
type TimeExpression struct {
	BaseNode
	Value string // e.g., "10s", "5m", "1h"
}

func (te *TimeExpression) String() string  { return "TimeExpression(" + te.Value + ")" }
func (te *TimeExpression) expressionNode() {}

// IPExpression represents IP address literals
type IPExpression struct {
	BaseNode
	Value string // e.g., "192.168.1.1", "::1"
}

func (ie *IPExpression) String() string  { return "IPExpression(" + ie.Value + ")" }
func (ie *IPExpression) expressionNode() {}

// ErrorExpression represents a placeholder expression used during error recovery
type ErrorExpression struct {
	BaseNode
	Message string
}

func (ee *ErrorExpression) String() string  { return "ErrorExpression(" + ee.Message + ")" }
func (ee *ErrorExpression) expressionNode() {}
