package ast

import (
	"github.com/varnish/vclparser/lexer"
)

// Node represents any node in the AST
type Node interface {
	String() string
	Start() lexer.Position
	End() lexer.Position
}

// BaseNode provides common functionality for all AST nodes
type BaseNode struct {
	StartPos lexer.Position
	EndPos   lexer.Position
}

func (b BaseNode) Start() lexer.Position { return b.StartPos }
func (b BaseNode) End() lexer.Position   { return b.EndPos }

// Program represents the root of a VCL AST
type Program struct {
	BaseNode
	VCLVersion   *VCLVersionDecl
	Declarations []Declaration
}

func (p *Program) String() string { return "Program" }

// Declaration represents any top-level declaration
type Declaration interface {
	Node
	declarationNode()
}

// Statement represents any statement within a subroutine
type Statement interface {
	Node
	statementNode()
}

// Expression represents any expression
type Expression interface {
	Node
	expressionNode()
}

// VCLVersionDecl represents a VCL version declaration (e.g., "vcl 4.0;")
type VCLVersionDecl struct {
	BaseNode
	Version string // e.g., "4.0", "4.1"
}

func (v *VCLVersionDecl) String() string   { return "VCLVersionDecl(" + v.Version + ")" }
func (v *VCLVersionDecl) declarationNode() {}

// ImportDecl represents an import declaration
type ImportDecl struct {
	BaseNode
	Module string
	Alias  string // optional alias
}

func (i *ImportDecl) String() string   { return "ImportDecl(" + i.Module + ")" }
func (i *ImportDecl) declarationNode() {}

// IncludeDecl represents an include declaration
type IncludeDecl struct {
	BaseNode
	Path string
}

func (i *IncludeDecl) String() string   { return "IncludeDecl(" + i.Path + ")" }
func (i *IncludeDecl) declarationNode() {}

// BackendDecl represents a backend declaration
type BackendDecl struct {
	BaseNode
	Name       string
	Properties []*BackendProperty
}

func (b *BackendDecl) String() string   { return "BackendDecl(" + b.Name + ")" }
func (b *BackendDecl) declarationNode() {}

// BackendProperty represents a property within a backend declaration
type BackendProperty struct {
	BaseNode
	Name  string
	Value Expression
}

func (bp *BackendProperty) String() string { return "BackendProperty(" + bp.Name + ")" }

// ProbeDecl represents a probe declaration
type ProbeDecl struct {
	BaseNode
	Name       string
	Properties []*ProbeProperty
}

func (p *ProbeDecl) String() string   { return "ProbeDecl(" + p.Name + ")" }
func (p *ProbeDecl) declarationNode() {}

// ProbeProperty represents a property within a probe declaration
type ProbeProperty struct {
	BaseNode
	Name  string
	Value Expression
}

func (pp *ProbeProperty) String() string { return "ProbeProperty(" + pp.Name + ")" }

// ACLDecl represents an ACL declaration
type ACLDecl struct {
	BaseNode
	Name    string
	Entries []*ACLEntry
}

func (a *ACLDecl) String() string   { return "ACLDecl(" + a.Name + ")" }
func (a *ACLDecl) declarationNode() {}

// ACLEntry represents an entry in an ACL
type ACLEntry struct {
	BaseNode
	Negated bool
	Network Expression // IP address or CIDR
}

func (ae *ACLEntry) String() string { return "ACLEntry" }

// SubDecl represents a subroutine declaration
type SubDecl struct {
	BaseNode
	Name string
	Body *BlockStatement
}

func (s *SubDecl) String() string   { return "SubDecl(" + s.Name + ")" }
func (s *SubDecl) declarationNode() {}

// Identifier represents an identifier
type Identifier struct {
	BaseNode
	Name string
}

func (i *Identifier) String() string  { return "Identifier(" + i.Name + ")" }
func (i *Identifier) expressionNode() {}

// StringLiteral represents a string literal
type StringLiteral struct {
	BaseNode
	Value string
}

func (s *StringLiteral) String() string  { return "StringLiteral(" + s.Value + ")" }
func (s *StringLiteral) expressionNode() {}

// IntegerLiteral represents an integer literal
type IntegerLiteral struct {
	BaseNode
	Value int64
}

func (i *IntegerLiteral) String() string  { return "IntegerLiteral" }
func (i *IntegerLiteral) expressionNode() {}

// FloatLiteral represents a floating-point literal
type FloatLiteral struct {
	BaseNode
	Value float64
}

func (f *FloatLiteral) String() string  { return "FloatLiteral" }
func (f *FloatLiteral) expressionNode() {}

// BooleanLiteral represents a boolean literal
type BooleanLiteral struct {
	BaseNode
	Value bool
}

func (b *BooleanLiteral) String() string  { return "BooleanLiteral" }
func (b *BooleanLiteral) expressionNode() {}

// DurationLiteral represents a duration literal (e.g., "10s", "5m")
type DurationLiteral struct {
	BaseNode
	Value string // The raw string representation
}

func (d *DurationLiteral) String() string  { return "DurationLiteral(" + d.Value + ")" }
func (d *DurationLiteral) expressionNode() {}

// VCLType represents the types available in VCL
type VCLType int

const (
	TypeString VCLType = iota
	TypeInt
	TypeFloat
	TypeBool
	TypeTime
	TypeDuration
	TypeIP
	TypeHeader
	TypeBackend
	TypeACL
	TypeProbe
	TypeVoid
)

func (t VCLType) String() string {
	switch t {
	case TypeString:
		return "STRING"
	case TypeInt:
		return "INT"
	case TypeFloat:
		return "REAL"
	case TypeBool:
		return "BOOL"
	case TypeTime:
		return "TIME"
	case TypeDuration:
		return "DURATION"
	case TypeIP:
		return "IP"
	case TypeHeader:
		return "HEADER"
	case TypeBackend:
		return "BACKEND"
	case TypeACL:
		return "ACL"
	case TypeProbe:
		return "PROBE"
	case TypeVoid:
		return "VOID"
	default:
		return "UNKNOWN"
	}
}
