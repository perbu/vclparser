package main

import (
	"fmt"
	"os"

	"github.com/perbu/vclparser/pkg/analyzer"
	"github.com/perbu/vclparser/pkg/parser"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run examples/metadata_validation_demo.go <vcl_file>")
		fmt.Println("\nExample VCL files to test:")
		fmt.Println("1. Valid VCL 4.1:")
		fmt.Println(`vcl 4.1;
sub vcl_recv { set req.url = "/test"; return (hash); }
sub vcl_backend_response { set beresp.ttl = 300s; return (deliver); }`)

		fmt.Println("\n2. Invalid VCL with version issues:")
		fmt.Println(`vcl 4.0;
sub vcl_backend_response { set beresp.proto = "HTTP/1.1"; return (deliver); }`)

		fmt.Println("\n3. Invalid VCL with access issues:")
		fmt.Println(`vcl 4.1;
sub vcl_recv { set beresp.status = 200; return (deliver); }`)
		os.Exit(1)
	}

	filename := os.Args[1]
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}

	// Parse the VCL code
	fmt.Printf("Parsing VCL file: %s\n", filename)
	program, err := parser.Parse(string(content), filename)
	if err != nil {
		fmt.Printf("Parse error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully parsed VCL file\n")

	// Perform metadata-driven validation
	fmt.Printf("\n🔍 Running metadata-driven validation...\n")
	analyzer := analyzer.NewAnalyzer(nil) // nil VMOD registry for this demo
	errors := analyzer.Analyze(program)

	if len(errors) == 0 {
		fmt.Printf("✅ All validation checks passed!\n")
		fmt.Printf("\nValidation checks performed:\n")
		fmt.Printf("  • Return action validation (40 tokens from metadata)\n")
		fmt.Printf("  • Variable access validation (126 variables from metadata)\n")
		fmt.Printf("  • VCL version compatibility (version_low/version_high)\n")
		fmt.Printf("  • Lexer token validation (40 tokens from metadata)\n")
	} else {
		fmt.Printf("❌ Found %d validation error(s):\n", len(errors))
		for i, error := range errors {
			fmt.Printf("  %d. %s\n", i+1, error)
		}
	}

	// Show VCL version info if available
	if program.VCLVersion != nil {
		fmt.Printf("\n📋 VCL Version: %s\n", program.VCLVersion.Version)
	} else {
		fmt.Printf("\n⚠️  No VCL version specified\n")
	}

	fmt.Printf("\n🎯 Metadata-driven validation complete!\n")
}
