package renderer

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/perbu/vclparser/pkg/ast"
)

// VCLRenderer implements a visitor that renders AST nodes back to VCL source code
type VCLRenderer struct {
	ast.BaseVisitor
	builder strings.Builder
	indent  int
}

// New creates a new VCL renderer
func New() *VCLRenderer {
	return &VCLRenderer{
		indent: 0,
	}
}

// Render renders an AST program to VCL source code
func Render(program *ast.Program) string {
	renderer := New()
	ast.Accept(program, renderer)
	return renderer.builder.String()
}

// Helper methods for writing and indentation

func (r *VCLRenderer) write(s string) {
	r.builder.WriteString(s)
}

func (r *VCLRenderer) writeLine(s string) {
	r.writeIndent()
	r.builder.WriteString(s)
	r.builder.WriteString("\n")
}

func (r *VCLRenderer) writeIndent() {
	for i := 0; i < r.indent; i++ {
		r.builder.WriteString("    ")
	}
}

func (r *VCLRenderer) newline() {
	r.builder.WriteString("\n")
}

func (r *VCLRenderer) indentInc() {
	r.indent++
}

func (r *VCLRenderer) indentDec() {
	r.indent--
}

// Visitor implementations

func (r *VCLRenderer) VisitProgram(node *ast.Program) interface{} {
	// Render VCL version
	if node.VCLVersion != nil {
		ast.Accept(node.VCLVersion, r)
		r.newline()
	}

	// Render declarations
	for i, decl := range node.Declarations {
		if i > 0 {
			r.newline()
		}
		ast.Accept(decl, r)
	}

	return nil
}

func (r *VCLRenderer) VisitVCLVersionDecl(node *ast.VCLVersionDecl) interface{} {
	r.writeLine(fmt.Sprintf("vcl %s;", node.Version))
	return nil
}

func (r *VCLRenderer) VisitImportDecl(node *ast.ImportDecl) interface{} {
	if node.Alias != "" {
		r.writeLine(fmt.Sprintf("import %s as %s;", node.Module, node.Alias))
	} else {
		r.writeLine(fmt.Sprintf("import %s;", node.Module))
	}
	return nil
}

func (r *VCLRenderer) VisitIncludeDecl(node *ast.IncludeDecl) interface{} {
	r.writeLine(fmt.Sprintf("include \"%s\";", node.Path))
	return nil
}

func (r *VCLRenderer) VisitBackendDecl(node *ast.BackendDecl) interface{} {
	r.writeLine(fmt.Sprintf("backend %s {", node.Name))
	r.indentInc()

	for _, prop := range node.Properties {
		r.writeIndent()
		r.write(fmt.Sprintf(".%s = ", prop.Name))
		ast.Accept(prop.Value, r)
		r.write(";")
		r.newline()
	}

	r.indentDec()
	r.writeLine("}")
	return nil
}

func (r *VCLRenderer) VisitProbeDecl(node *ast.ProbeDecl) interface{} {
	r.writeLine(fmt.Sprintf("probe %s {", node.Name))
	r.indentInc()

	for _, prop := range node.Properties {
		r.writeIndent()
		r.write(fmt.Sprintf(".%s = ", prop.Name))
		ast.Accept(prop.Value, r)
		r.write(";")
		r.newline()
	}

	r.indentDec()
	r.writeLine("}")
	return nil
}

func (r *VCLRenderer) VisitACLDecl(node *ast.ACLDecl) interface{} {
	r.writeLine(fmt.Sprintf("acl %s {", node.Name))
	r.indentInc()

	for _, entry := range node.Entries {
		r.writeIndent()
		if entry.Negated {
			r.write("!")
		}
		ast.Accept(entry.Network, r)
		r.write(";")
		r.newline()
	}

	r.indentDec()
	r.writeLine("}")
	return nil
}

func (r *VCLRenderer) VisitSubDecl(node *ast.SubDecl) interface{} {
	r.writeLine(fmt.Sprintf("sub %s {", node.Name))
	r.indentInc()

	if node.Body != nil {
		for _, stmt := range node.Body.Statements {
			ast.Accept(stmt, r)
		}
	}

	r.indentDec()
	r.writeLine("}")
	return nil
}

// Statement visitors

func (r *VCLRenderer) VisitBlockStatement(node *ast.BlockStatement) interface{} {
	r.write("{")
	r.newline()
	r.indentInc()

	for _, stmt := range node.Statements {
		ast.Accept(stmt, r)
	}

	r.indentDec()
	r.writeIndent()
	r.write("}")
	return nil
}

func (r *VCLRenderer) VisitExpressionStatement(node *ast.ExpressionStatement) interface{} {
	r.writeIndent()
	ast.Accept(node.Expression, r)
	r.write(";")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitIfStatement(node *ast.IfStatement) interface{} {
	return r.renderIfStatement(node, true)
}

func (r *VCLRenderer) renderIfStatement(node *ast.IfStatement, writeIndent bool) interface{} {
	if writeIndent {
		r.writeIndent()
	}
	r.write("if (")
	ast.Accept(node.Condition, r)
	r.write(") ")

	// Handle single-statement then
	if _, isBlock := node.Then.(*ast.BlockStatement); isBlock {
		ast.Accept(node.Then, r)
		r.newline()
	} else {
		r.newline()
		r.indentInc()
		ast.Accept(node.Then, r)
		r.indentDec()
	}

	// Handle else
	if node.Else != nil {
		r.writeIndent()
		if ifStmt, isIf := node.Else.(*ast.IfStatement); isIf {
			// else if case
			r.write("else ")
			r.renderIfStatement(ifStmt, false)
		} else if _, isBlock := node.Else.(*ast.BlockStatement); isBlock {
			r.write("else ")
			ast.Accept(node.Else, r)
			r.newline()
		} else {
			r.write("else")
			r.newline()
			r.indentInc()
			ast.Accept(node.Else, r)
			r.indentDec()
		}
	}

	return nil
}

func (r *VCLRenderer) VisitSetStatement(node *ast.SetStatement) interface{} {
	r.writeIndent()
	r.write("set ")
	ast.Accept(node.Variable, r)
	r.write(fmt.Sprintf(" %s ", node.Operator))
	ast.Accept(node.Value, r)
	r.write(";")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitUnsetStatement(node *ast.UnsetStatement) interface{} {
	r.writeIndent()
	r.write("unset ")
	ast.Accept(node.Variable, r)
	r.write(";")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitCallStatement(node *ast.CallStatement) interface{} {
	r.writeIndent()
	r.write("call ")
	ast.Accept(node.Function, r)
	r.write(";")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitReturnStatement(node *ast.ReturnStatement) interface{} {
	r.writeIndent()
	r.write("return(")
	ast.Accept(node.Action, r)
	r.write(");")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitSyntheticStatement(node *ast.SyntheticStatement) interface{} {
	r.writeIndent()
	r.write("synthetic(")
	ast.Accept(node.Response, r)
	r.write(");")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitErrorStatement(node *ast.ErrorStatement) interface{} {
	r.writeIndent()
	r.write("error")
	if node.Code != nil || node.Response != nil {
		r.write("(")
		if node.Code != nil {
			ast.Accept(node.Code, r)
		}
		if node.Response != nil {
			if node.Code != nil {
				r.write(", ")
			}
			ast.Accept(node.Response, r)
		}
		r.write(")")
	}
	r.write(";")
	r.newline()
	return nil
}

func (r *VCLRenderer) VisitRestartStatement(node *ast.RestartStatement) interface{} {
	r.writeLine("restart;")
	return nil
}

func (r *VCLRenderer) VisitCSourceStatement(node *ast.CSourceStatement) interface{} {
	r.writeLine("C{")
	// Write C code as-is without indentation
	r.write(node.Code)
	r.writeLine("}C")
	return nil
}

func (r *VCLRenderer) VisitNewStatement(node *ast.NewStatement) interface{} {
	r.writeIndent()
	r.write("new ")
	ast.Accept(node.Name, r)
	r.write(" = ")
	ast.Accept(node.Constructor, r)
	r.write(";")
	r.newline()
	return nil
}

// Expression visitors

func (r *VCLRenderer) VisitBinaryExpression(node *ast.BinaryExpression) interface{} {
	ast.Accept(node.Left, r)
	r.write(fmt.Sprintf(" %s ", node.Operator))
	ast.Accept(node.Right, r)
	return nil
}

func (r *VCLRenderer) VisitUnaryExpression(node *ast.UnaryExpression) interface{} {
	r.write(node.Operator)
	ast.Accept(node.Operand, r)
	return nil
}

func (r *VCLRenderer) VisitCallExpression(node *ast.CallExpression) interface{} {
	ast.Accept(node.Function, r)
	r.write("(")

	// Render positional arguments
	for i, arg := range node.Arguments {
		if i > 0 {
			r.write(", ")
		}
		ast.Accept(arg, r)
	}

	// Render named arguments (sorted by key for consistency)
	if len(node.NamedArguments) > 0 {
		// Sort keys for deterministic output
		keys := make([]string, 0, len(node.NamedArguments))
		for k := range node.NamedArguments {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			if len(node.Arguments) > 0 || key != keys[0] {
				r.write(", ")
			}
			r.write(fmt.Sprintf("%s = ", key))
			ast.Accept(node.NamedArguments[key], r)
		}
	}

	r.write(")")
	return nil
}

func (r *VCLRenderer) VisitMemberExpression(node *ast.MemberExpression) interface{} {
	ast.Accept(node.Object, r)
	r.write(".")
	ast.Accept(node.Property, r)
	return nil
}

func (r *VCLRenderer) VisitIndexExpression(node *ast.IndexExpression) interface{} {
	ast.Accept(node.Object, r)
	r.write("[")
	ast.Accept(node.Index, r)
	r.write("]")
	return nil
}

func (r *VCLRenderer) VisitParenthesizedExpression(node *ast.ParenthesizedExpression) interface{} {
	r.write("(")
	ast.Accept(node.Expression, r)
	r.write(")")
	return nil
}

func (r *VCLRenderer) VisitRegexMatchExpression(node *ast.RegexMatchExpression) interface{} {
	ast.Accept(node.Left, r)
	r.write(fmt.Sprintf(" %s ", node.Operator))
	ast.Accept(node.Right, r)
	return nil
}

func (r *VCLRenderer) VisitAssignmentExpression(node *ast.AssignmentExpression) interface{} {
	ast.Accept(node.Left, r)
	r.write(fmt.Sprintf(" %s ", node.Operator))
	ast.Accept(node.Right, r)
	return nil
}

func (r *VCLRenderer) VisitUpdateExpression(node *ast.UpdateExpression) interface{} {
	if node.Prefix {
		r.write(node.Operator)
		ast.Accept(node.Operand, r)
	} else {
		ast.Accept(node.Operand, r)
		r.write(node.Operator)
	}
	return nil
}

func (r *VCLRenderer) VisitArrayExpression(node *ast.ArrayExpression) interface{} {
	r.write("[")
	for i, elem := range node.Elements {
		if i > 0 {
			r.write(", ")
		}
		ast.Accept(elem, r)
	}
	r.write("]")
	return nil
}

func (r *VCLRenderer) VisitObjectExpression(node *ast.ObjectExpression) interface{} {
	r.write("{")
	for i, prop := range node.Properties {
		if i > 0 {
			r.write(", ")
		}
		ast.Accept(prop.Key, r)
		r.write(": ")
		ast.Accept(prop.Value, r)
	}
	r.write("}")
	return nil
}

func (r *VCLRenderer) VisitVariableExpression(node *ast.VariableExpression) interface{} {
	r.write(node.Name)
	return nil
}

func (r *VCLRenderer) VisitTimeExpression(node *ast.TimeExpression) interface{} {
	r.write(node.Value)
	return nil
}

func (r *VCLRenderer) VisitIPExpression(node *ast.IPExpression) interface{} {
	r.write(fmt.Sprintf("\"%s\"", node.Value))
	return nil
}

// Literal visitors

func (r *VCLRenderer) VisitIdentifier(node *ast.Identifier) interface{} {
	r.write(node.Name)
	return nil
}

func (r *VCLRenderer) VisitStringLiteral(node *ast.StringLiteral) interface{} {
	// Escape special characters in strings
	escaped := strings.ReplaceAll(node.Value, "\\", "\\\\")
	escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
	escaped = strings.ReplaceAll(escaped, "\n", "\\n")
	escaped = strings.ReplaceAll(escaped, "\r", "\\r")
	escaped = strings.ReplaceAll(escaped, "\t", "\\t")
	r.write(fmt.Sprintf("\"%s\"", escaped))
	return nil
}

func (r *VCLRenderer) VisitIntegerLiteral(node *ast.IntegerLiteral) interface{} {
	r.write(strconv.FormatInt(node.Value, 10))
	return nil
}

func (r *VCLRenderer) VisitFloatLiteral(node *ast.FloatLiteral) interface{} {
	r.write(strconv.FormatFloat(node.Value, 'f', -1, 64))
	return nil
}

func (r *VCLRenderer) VisitBooleanLiteral(node *ast.BooleanLiteral) interface{} {
	if node.Value {
		r.write("true")
	} else {
		r.write("false")
	}
	return nil
}

func (r *VCLRenderer) VisitDurationLiteral(node *ast.DurationLiteral) interface{} {
	r.write(node.Value)
	return nil
}
