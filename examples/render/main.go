package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/include"
	"github.com/perbu/vclparser/pkg/parser"
	"github.com/perbu/vclparser/pkg/renderer"
)

func main() {
	var (
		inputFile  = flag.String("file", "", "Input VCL file to render")
		outputFile = flag.String("output", "", "Output file (default: stdout)")
		resolveInc = flag.Bool("resolve-includes", false, "Resolve include statements before rendering")
		basePath   = flag.String("base", ".", "Base path for resolving includes")
		showHelp   = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *showHelp || *inputFile == "" {
		fmt.Println("VCL Renderer - Render VCL AST back to source code")
		fmt.Println()
		fmt.Println("Usage:")
		fmt.Printf("  %s -file <input.vcl> [options]\n", os.Args[0])
		fmt.Println()
		fmt.Println("Options:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Printf("  %s -file main.vcl\n", os.Args[0])
		fmt.Printf("  %s -file main.vcl -output formatted.vcl\n", os.Args[0])
		fmt.Printf("  %s -file main.vcl -resolve-includes -base /etc/varnish\n", os.Args[0])
		fmt.Println()
		fmt.Println("Description:")
		fmt.Println("  This tool parses VCL files and renders them back to source code.")
		fmt.Println("  It can be used for:")
		fmt.Println("  - Formatting VCL files")
		fmt.Println("  - Validating VCL syntax")
		fmt.Println("  - Merging include statements into a single file")
		fmt.Println("  - Normalizing VCL code style")
		os.Exit(0)
	}

	// Read the input file
	content, err := os.ReadFile(*inputFile)
	if err != nil {
		log.Fatalf("Error reading file %s: %v", *inputFile, err)
	}

	var program *ast.Program

	if *resolveInc {
		// Parse with include resolution
		resolver := include.NewResolver(
			include.WithBasePath(*basePath),
		)
		program, err = resolver.ResolveFile(*inputFile)
		if err != nil {
			log.Fatalf("Error resolving includes: %v", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Parsed %s with includes resolved\n", *inputFile)
	} else {
		// Parse without include resolution
		program, err = parser.Parse(string(content), *inputFile)
		if err != nil {
			log.Fatalf("Parse error: %v", err)
		}
		fmt.Fprintf(os.Stderr, "✓ Parsed %s\n", *inputFile)
	}

	// Render the AST back to VCL
	rendered := renderer.Render(program)

	// Write output
	if *outputFile != "" {
		err = os.WriteFile(*outputFile, []byte(rendered), 0644)
		if err != nil {
			log.Fatalf("Error writing output file %s: %v", *outputFile, err)
		}
		fmt.Fprintf(os.Stderr, "✓ Rendered VCL written to %s\n", *outputFile)

		// Show statistics
		showStatistics(program, *outputFile)
	} else {
		// Write to stdout
		fmt.Print(rendered)

		// Show statistics to stderr
		fmt.Fprintln(os.Stderr)
		showStatistics(program, "")
	}
}

func showStatistics(program *ast.Program, outputFile string) {
	backends := 0
	subroutines := 0
	acls := 0
	probes := 0
	imports := 0
	includes := 0

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
		case *ast.ImportDecl:
			imports++
		case *ast.IncludeDecl:
			includes++
		}
	}

	fmt.Fprintf(os.Stderr, "\nStatistics:\n")
	if program.VCLVersion != nil {
		fmt.Fprintf(os.Stderr, "  VCL Version:  %s\n", program.VCLVersion.Version)
	}
	fmt.Fprintf(os.Stderr, "  Backends:     %d\n", backends)
	fmt.Fprintf(os.Stderr, "  Subroutines:  %d\n", subroutines)
	fmt.Fprintf(os.Stderr, "  ACLs:         %d\n", acls)
	fmt.Fprintf(os.Stderr, "  Probes:       %d\n", probes)
	fmt.Fprintf(os.Stderr, "  Imports:      %d\n", imports)
	fmt.Fprintf(os.Stderr, "  Includes:     %d\n", includes)
}
