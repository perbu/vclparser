# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

VCL Parser is a Go implementation of a parser for VCL (Varnish Configuration Language) that generates Abstract Syntax Trees (AST). It supports VCL 4.0 and 4.1 syntax with complete lexical analysis, recursive descent parsing, type-safe AST representation, and semantic analysis.

## Architecture

The codebase follows a clean separation of concerns across four main packages:

### Core Parser Pipeline
- **pkg/lexer/** - Tokenizes VCL source code into lexical tokens (`lexer.go`, `token.go`)
- **pkg/parser/** - Recursive descent parser that converts tokens to AST (`parser.go`, `expressions.go`, `statements.go`, `declarations.go`)
- **pkg/ast/** - AST node definitions and visitor pattern implementation (`node.go`, `expressions.go`, `statements.go`, `visitor.go`)
- **pkg/renderer/** - Renders AST back to VCL source code (`renderer.go`)
- **pkg/types/** - Type system and symbol table for semantic analysis (`types.go`, `symbol_table.go`)
- **pkg/include/** - Include statement resolution with circular dependency detection (`resolver.go`, `api.go`, `errors.go`)

### Key Design Patterns
- **Visitor Pattern**: Used for AST traversal - see `pkg/ast/visitor.go` for base implementation
- **Recursive Descent Parsing**: Parser methods correspond to grammar productions
- **Error Recovery**: Parser continues after errors to report multiple issues
- **Symbol Tables**: Track variable scope and built-in VCL functions/variables

### Entry Points
- **parser.Parse(input, filename, opts...)** - Main parsing function with functional options
- **parser.New(lexer, input, filename, opts...)** - Create parser instance with options
- **renderer.Render(program)** - Renders an AST back to VCL source code
- **include.ResolveFile(filename)** - Alternative API for include resolution (legacy)
- **examples/parse/main.go** - CLI tool demonstrating basic parser usage with pretty-printing and JSON export
- **examples/includes/main.go** - CLI tool demonstrating both parser and include resolver APIs
- **examples/render/main.go** - CLI tool for rendering/formatting VCL files with optional include resolution

## Common Commands

### Testing
```bash
go test ./...                    # Run all tests
go test ./parser                 # Test specific package
go test -v ./parser              # Verbose test output
go test -run TestName ./parser   # Run specific test
```

### Building and Running
```bash
# Parser examples
go build ./examples/parse/main.go              # Build basic parser example
go run ./examples/parse/main.go file.vcl       # Parse and pretty-print VCL file
go run ./examples/parse/main.go file.vcl --json # Export AST as JSON

# Include resolution examples
go build ./examples/includes/main.go           # Build include-aware parser
go run ./examples/includes/main.go -file main.vcl  # Parse with includes
go run ./examples/includes/main.go -file main.vcl -base /etc/varnish  # Set base path
go run ./examples/includes/main.go -file main.vcl -json > merged.json # Export merged AST

# VCL Renderer examples
go build ./examples/render/main.go             # Build VCL renderer
go run ./examples/render/main.go -file input.vcl  # Format VCL file
go run ./examples/render/main.go -file input.vcl -output formatted.vcl  # Save to file
go run ./examples/render/main.go -file main.vcl -resolve-includes  # Merge includes into single file
```

### Code Quality
```bash
go fmt ./...                     # Format all Go files
go vet ./...                     # Static analysis
go mod tidy                      # Clean up module dependencies
```

### Development Workflow
```bash
go build -o parse_vcl ./examples/parse_vcl.go  # Build executable
./parse_vcl tests/testdata/simple.vcl          # Test with sample VCL
```

## VCL Language Features

The parser supports the complete VCL specification including:
- Version declarations (`vcl 4.0;`, `vcl 4.1;`)
- Backend definitions with properties
- Subroutine definitions (`sub vcl_recv { }`)
- **Multiple subroutine definitions** - Same subroutine can be defined multiple times, bodies are merged in order
- Control flow statements (if/else, return, call)
- Variable assignments (set, unset)
- Expression parsing with proper operator precedence
- Built-in VCL variables (`req.method`, `beresp.status`, etc.)
- **Named parameter syntax for VMOD functions** (`headerplus.as_list(NAME, ";", name_case = LOWER)`)
- **Include statements with recursive resolution** (`include "backends.vcl";`)
- C-code blocks (`C{ }C`)
- Access Control Lists (ACLs)
- Probe definitions

## Known Limitations

Based on TODO.md, current limitations include:
- **Return Statement Action Keywords**: Parser requires parentheses around return actions (`return (lookup);` works, but `return lookup;` doesn't) - **DESIGN DECISION**: We will maintain this requirement for clarity and consistency
- **Object Literal Parsing**: Complex backend properties like inline probes are not fully supported
- **Advanced Expressions**: Some complex expressions may not parse correctly

## Recent Improvements (2025)

- ✅ **Functional Options Pattern**: Idiomatic Go API for parser configuration
  - Replaced Config struct with functional options
  - Single, clean API: `parser.Parse()` and `parser.New()` with variadic options
  - Options: `WithMaxErrors()`, `WithDisableInlineC()`, `WithAllowMissingVersion()`, etc.
  - Breaking change from previous Config-based API
  - Examples: `parser.Parse(content, "file.vcl", parser.WithMaxErrors(10))`

- ✅ **Integrated Include Resolution**: Parser can automatically resolve includes
  - New options: `WithResolveIncludes(basePath)` and `WithIncludeMaxDepth(depth)`
  - Include resolution logic integrated into parser package (avoids circular dependencies)
  - Simplifies common use case: parse + resolve in one call
  - Alternative: `pkg/include` package still available for advanced use cases
  - Examples: `parser.Parse(content, "main.vcl", parser.WithResolveIncludes("/etc/varnish"))`

- ✅ **Named Parameter Syntax**: Complete support for named arguments in VMOD function calls
  - Two-phase parsing (positional then named arguments)
  - Duplicate and unknown parameter detection
  - Compatible with varnishd argument parsing behavior
  - Examples: `headerplus.as_list(NAME, ";", name_case = LOWER)`

- ✅ **VCL Renderer**: Convert AST back to VCL source code
  - Visitor-based implementation for clean AST traversal
  - Proper indentation and formatting
  - Preserves semantic meaning while normalizing style
  - Supports all VCL constructs including named parameters
  - Use cases: code formatting, include merging, AST manipulation
  - Examples: `renderer.Render(program)`

## Include Statement Handling

VCL include statements can be handled in two ways:

### Approach 1: Integrated Parser API (Recommended)

Use parser options for automatic include resolution:

```go
// Read file and parse with includes resolved
content, _ := os.ReadFile("main.vcl")
program, err := parser.Parse(string(content), "main.vcl",
    parser.WithResolveIncludes("/etc/varnish"),
    parser.WithIncludeMaxDepth(10),
)
```

This is the simplest approach for most use cases. The parser automatically:
- Recognizes `include "file.vcl";` statements
- Recursively reads and parses included files
- Merges all declarations into a single AST
- Handles errors: circular dependencies, missing files, syntax errors, depth limits

### Approach 2: Separate Include Resolver (Advanced)

For more control, use the `pkg/include` package directly:

```go
// Parse first (creates IncludeDecl nodes)
program, err := parser.Parse(content, "main.vcl")

// Resolve later
resolver := include.NewResolver(
    include.WithBasePath("/etc/varnish"),
    include.WithMaxDepth(5),
)
resolved, err := resolver.ResolveProgram(program)
```

Or use the convenience function:

```go
// Parse and resolve in one call
program, err := include.ResolveFile("main.vcl")
```

This approach provides a two-phase workflow:
- **Phase 1**: Parser creates `IncludeDecl` AST nodes (preserves syntax tree)
- **Phase 2**: Resolver processes nodes and merges included files

### Error Handling
- `CircularIncludeError` - Detects and reports include loops with full chain
- `MaxDepthError` - Prevents excessive nesting (default limit: 10)
- `FileNotFoundError` - Missing include files with path context
- `ParseError` - Syntax errors in included files

### Differences from Varnish
- Varnish resolves includes during parsing (like C preprocessor)
- This parser uses post-parse resolution for cleaner architecture
- Both approaches produce equivalent final ASTs for valid VCL

## Testing Strategy

- **tests/testdata/** contains sample VCL files for testing
- **tests/testdata/includes/** contains comprehensive include resolution test cases
- Each parser component has dedicated test files (*_test.go)
- Tests cover both valid VCL parsing and error recovery scenarios
- Include resolver has extensive tests for edge cases and error conditions

## Varnishd Reference Tools

The **varnishd/** directory contains code generation tools lifted from the Varnish HTTP accelerator project that serve as reference material for understanding VCL language implementation:

- **generate.py** - Python script that generates C code definitions for VCL lexer, parser, and type system
- **vmodtool.py** - VMOD (Varnish Module) build tool that generates C interfaces and documentation from `.vcc` specification files
- run "make" after changes to see if the code is good