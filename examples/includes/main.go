package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/include"
)

func main() {
	var (
		filename   = flag.String("file", "", "VCL file to parse (required)")
		basePath   = flag.String("base", "", "Base path for resolving relative includes (defaults to file's directory)")
		outputJSON = flag.Bool("json", false, "Output AST as JSON instead of pretty-printing")
		maxDepth   = flag.Int("max-depth", 10, "Maximum include depth")
		showHelp   = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *showHelp || *filename == "" {
		printHelp()
		if *filename == "" {
			os.Exit(1)
		}
		return
	}

	// Determine base path
	resolveBasePath := *basePath
	if resolveBasePath == "" {
		resolveBasePath = filepath.Dir(*filename)
	}

	fmt.Printf("Parsing VCL file: %s\n", *filename)
	fmt.Printf("Base path: %s\n", resolveBasePath)
	fmt.Printf("Max depth: %d\n", *maxDepth)
	fmt.Println()

	// Create resolver with options
	resolver := include.NewResolver(
		include.WithBasePath(resolveBasePath),
		include.WithMaxDepth(*maxDepth),
	)

	// Parse and resolve includes
	program, err := resolver.ResolveFile(filepath.Base(*filename))
	if err != nil {
		handleError(err)
		os.Exit(1)
	}

	if *outputJSON {
		outputJSONFormat(program)
	} else {
		outputPrettyFormat(program)
	}
}

//nolint:nilaway
func printHelp() {
	fmt.Println("VCL Parser with Include Resolution")
	fmt.Println()
	fmt.Println("This tool demonstrates how to parse VCL files with include statements.")
	fmt.Println("It resolves all includes recursively and outputs the merged AST.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Printf("  %s -file <vcl-file> [options]\n", os.Args[0])
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Printf("  %s -file main.vcl\n", os.Args[0])
	fmt.Printf("  %s -file main.vcl -base /etc/varnish\n", os.Args[0])
	fmt.Printf("  %s -file main.vcl -json > ast.json\n", os.Args[0])
	fmt.Printf("  %s -file main.vcl -max-depth 5\n", os.Args[0])
	fmt.Println()
	fmt.Println("The tool automatically resolves include statements and merges all")
	fmt.Println("declarations into a single AST. Include paths are resolved relative")
	fmt.Println("to the base path (defaults to the directory of the main file).")
}

func handleError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)

	// Provide specific help for different error types
	switch e := err.(type) {
	case *include.CircularIncludeError:
		fmt.Fprintf(os.Stderr, "\nCircular include detected in chain: %v\n", e.Chain)
		fmt.Fprintf(os.Stderr, "Check your include statements for loops.\n")
	case *include.MaxDepthError:
		fmt.Fprintf(os.Stderr, "\nInclude depth limit exceeded. Current: %d, Limit: %d\n", e.Current, e.MaxDepth)
		fmt.Fprintf(os.Stderr, "Use -max-depth to increase the limit if needed.\n")
	case *include.FileNotFoundError:
		fmt.Fprintf(os.Stderr, "\nFile not found: %s\n", e.Path)
		if e.BasePath != "" {
			fmt.Fprintf(os.Stderr, "Base path: %s\n", e.BasePath)
		}
		fmt.Fprintf(os.Stderr, "Check the file path and use -base to set the correct base directory.\n")
	case *include.ParseError:
		fmt.Fprintf(os.Stderr, "\nSyntax error in file: %s\n", e.Path)
		fmt.Fprintf(os.Stderr, "Fix the VCL syntax in the included file.\n")
	}
}

func outputPrettyFormat(program *ast.Program) {
	fmt.Println("=== VCL Parse Results with Include Resolution ===")
	fmt.Println()

	if program.VCLVersion != nil {
		fmt.Printf("VCL Version: %s\n", program.VCLVersion.Version)
	}

	// Count declarations by type
	counts := make(map[string]int)
	for _, decl := range program.Declarations {
		switch decl.(type) {
		case *ast.BackendDecl:
			counts["backends"]++
		case *ast.SubDecl:
			counts["subroutines"]++
		case *ast.ACLDecl:
			counts["acls"]++
		case *ast.ImportDecl:
			counts["imports"]++
		case *ast.IncludeDecl:
			counts["includes"]++
		default:
			counts["other"]++
		}
	}

	fmt.Println("\nDeclaration Summary:")
	for declType, count := range counts {
		fmt.Printf("  %s: %d\n", declType, count)
	}

	if counts["includes"] > 0 {
		fmt.Printf("\nWarning: %d include declarations remain unresolved!\n", counts["includes"])
	} else {
		fmt.Println("\n✓ All include statements have been resolved")
	}

	fmt.Printf("\nTotal declarations: %d\n", len(program.Declarations))

	// Show detailed declarations
	fmt.Println("\n=== Declarations ===")
	for i, decl := range program.Declarations {
		fmt.Printf("%d. %s\n", i+1, decl.String())
	}
}

func outputJSONFormat(program *ast.Program) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(program); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to encode JSON: %v\n", err)
		os.Exit(1)
	}
}
