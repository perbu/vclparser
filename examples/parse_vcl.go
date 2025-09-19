package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/parser"
	"github.com/varnish/vclparser/types"
)

// PrettyPrinter implements a visitor that prints the AST in a readable format
type PrettyPrinter struct {
	ast.BaseVisitor
	indent int
}

func (pp *PrettyPrinter) print(format string, args ...interface{}) {
	for i := 0; i < pp.indent; i++ {
		fmt.Print("  ")
	}
	fmt.Printf(format, args...)
}

func (pp *PrettyPrinter) VisitProgram(node *ast.Program) interface{} {
	pp.print("Program:\n")
	pp.indent++

	if node.VCLVersion != nil {
		ast.Accept(node.VCLVersion, pp)
	}

	for _, decl := range node.Declarations {
		ast.Accept(decl, pp)
	}

	pp.indent--
	return nil
}

func (pp *PrettyPrinter) VisitVCLVersionDecl(node *ast.VCLVersionDecl) interface{} {
	pp.print("VCL Version: %s\n", node.Version)
	return nil
}

func (pp *PrettyPrinter) VisitBackendDecl(node *ast.BackendDecl) interface{} {
	pp.print("Backend: %s\n", node.Name)
	pp.indent++
	for _, prop := range node.Properties {
		pp.print("Property: %s = ", prop.Name)
		ast.Accept(prop.Value, pp)
		fmt.Println()
	}
	pp.indent--
	return nil
}

func (pp *PrettyPrinter) VisitSubDecl(node *ast.SubDecl) interface{} {
	pp.print("Subroutine: %s\n", node.Name)
	pp.indent++
	if node.Body != nil {
		ast.Accept(node.Body, pp)
	}
	pp.indent--
	return nil
}

func (pp *PrettyPrinter) VisitBlockStatement(node *ast.BlockStatement) interface{} {
	pp.print("Block:\n")
	pp.indent++
	for _, stmt := range node.Statements {
		ast.Accept(stmt, pp)
	}
	pp.indent--
	return nil
}

func (pp *PrettyPrinter) VisitIfStatement(node *ast.IfStatement) interface{} {
	pp.print("If: ")
	ast.Accept(node.Condition, pp)
	fmt.Println()
	pp.indent++
	if node.Then != nil {
		ast.Accept(node.Then, pp)
	}
	if node.Else != nil {
		pp.print("Else:\n")
		ast.Accept(node.Else, pp)
	}
	pp.indent--
	return nil
}

func (pp *PrettyPrinter) VisitBinaryExpression(node *ast.BinaryExpression) interface{} {
	fmt.Print("(")
	ast.Accept(node.Left, pp)
	fmt.Printf(" %s ", node.Operator)
	ast.Accept(node.Right, pp)
	fmt.Print(")")
	return nil
}

func (pp *PrettyPrinter) VisitMemberExpression(node *ast.MemberExpression) interface{} {
	ast.Accept(node.Object, pp)
	fmt.Print(".")
	ast.Accept(node.Property, pp)
	return nil
}

func (pp *PrettyPrinter) VisitIdentifier(node *ast.Identifier) interface{} {
	fmt.Print(node.Name)
	return nil
}

func (pp *PrettyPrinter) VisitStringLiteral(node *ast.StringLiteral) interface{} {
	fmt.Printf(`"%s"`, node.Value)
	return nil
}

// JSONExporter exports the AST as JSON
type JSONExporter struct {
	ast.BaseVisitor
}

func (je *JSONExporter) VisitProgram(node *ast.Program) interface{} {
	result := map[string]interface{}{
		"type": "Program",
	}

	if node.VCLVersion != nil {
		result["vclVersion"] = ast.Accept(node.VCLVersion, je)
	}

	declarations := make([]interface{}, len(node.Declarations))
	for i, decl := range node.Declarations {
		declarations[i] = ast.Accept(decl, je)
	}
	result["declarations"] = declarations

	return result
}

func (je *JSONExporter) VisitVCLVersionDecl(node *ast.VCLVersionDecl) interface{} {
	return map[string]interface{}{
		"type":    "VCLVersionDecl",
		"version": node.Version,
	}
}

func (je *JSONExporter) VisitBackendDecl(node *ast.BackendDecl) interface{} {
	properties := make([]interface{}, len(node.Properties))
	for i, prop := range node.Properties {
		properties[i] = map[string]interface{}{
			"name":  prop.Name,
			"value": ast.Accept(prop.Value, je),
		}
	}

	return map[string]interface{}{
		"type":       "BackendDecl",
		"name":       node.Name,
		"properties": properties,
	}
}

func (je *JSONExporter) VisitStringLiteral(node *ast.StringLiteral) interface{} {
	return map[string]interface{}{
		"type":  "StringLiteral",
		"value": node.Value,
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: parse_vcl <vcl-file> [--json]")
		os.Exit(1)
	}

	filename := os.Args[1]
	outputJSON := len(os.Args) > 2 && os.Args[2] == "--json"

	// Read the VCL file
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Parse the VCL
	program, err := parser.Parse(string(content), filename)
	if err != nil {
		log.Fatalf("Parse error: %v", err)
	}

	if outputJSON {
		// Export as JSON
		exporter := &JSONExporter{}
		result := ast.Accept(program, exporter)

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Fatalf("JSON marshal error: %v", err)
		}

		fmt.Println(string(jsonBytes))
	} else {
		// Pretty print the AST
		printer := &PrettyPrinter{}
		ast.Accept(program, printer)

		// Show some statistics
		fmt.Printf("\nParsing completed successfully!\n")
		fmt.Printf("VCL Version: %s\n", program.VCLVersion.Version)
		fmt.Printf("Declarations: %d\n", len(program.Declarations))

		// Count declaration types
		backends := 0
		subroutines := 0
		acls := 0
		probes := 0

		for _, decl := range program.Declarations {
			switch decl.(type) {
			case *ast.BackendDecl:
				backends++
			case *ast.SubDecl:
				subroutines++
			case *ast.ACLDecl:
				acls++
			case *ast.ProbeDecl:
				probes++
			}
		}

		fmt.Printf("  - Backends: %d\n", backends)
		fmt.Printf("  - Subroutines: %d\n", subroutines)
		fmt.Printf("  - ACLs: %d\n", acls)
		fmt.Printf("  - Probes: %d\n", probes)

		// Create symbol table for semantic analysis
		symbolTable := types.NewSymbolTable()
		if reqSymbol := symbolTable.Lookup("req.method"); reqSymbol != nil {
			fmt.Printf("\nBuilt-in symbols for req.method: %d\n", len(reqSymbol.Methods))
		} else {
			fmt.Printf("\nSymbol table created (req.method not found in built-ins)\n")
		}
	}
}
