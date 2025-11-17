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
- **pkg/types/** - Type system and symbol table for semantic analysis (`types.go`, `symbol_table.go`)
- **pkg/include/** - Include statement resolution with circular dependency detection (`resolver.go`, `api.go`, `errors.go`)

### Key Design Patterns
- **Visitor Pattern**: Used for AST traversal - see `pkg/ast/visitor.go` for base implementation
- **Recursive Descent Parsing**: Parser methods correspond to grammar productions
- **Error Recovery**: Parser continues after errors to report multiple issues
- **Symbol Tables**: Track variable scope and built-in VCL functions/variables

### Entry Points
- **parser.Parse(input, filename)** - Main parsing function that returns `*ast.Program`
- **include.ResolveFile(filename)** - Parse VCL file and resolve all include statements
- **examples/parse_vcl.go** - CLI tool demonstrating basic parser usage with pretty-printing and JSON export
- **examples/parse_with_includes.go** - CLI tool demonstrating include resolution with comprehensive options

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
go build ./examples/parse_vcl.go              # Build basic parser example
go run ./examples/parse_vcl.go file.vcl       # Parse and pretty-print VCL file
go run ./examples/parse_vcl.go file.vcl --json # Export AST as JSON

# Include resolution examples
go build ./examples/parse_with_includes.go    # Build include-aware parser
go run ./examples/parse_with_includes.go -file main.vcl  # Parse with includes
go run ./examples/parse_with_includes.go -file main.vcl -base /etc/varnish  # Set base path
go run ./examples/parse_with_includes.go -file main.vcl -json > merged.json # Export merged AST
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

- ✅ **Named Parameter Syntax**: Complete support for named arguments in VMOD function calls
  - Two-phase parsing (positional then named arguments)
  - Duplicate and unknown parameter detection
  - Compatible with varnishd argument parsing behavior
  - Examples: `headerplus.as_list(NAME, ";", name_case = LOWER)`

- ✅ **Include Statement Resolution**: Two-phase approach for handling VCL include statements
  - Core parser creates `IncludeDecl` AST nodes (preserves syntax tree structure)
  - Separate `pkg/include` package provides resolution with error handling
  - Circular dependency detection and configurable depth limits
  - Compatible with both single-file and multi-file VCL configurations
  - Examples: `include.ResolveFile("main.vcl")` or `include.ResolveProgram(ast)`

## Include Statement Handling

VCL include statements are handled using a **two-phase approach** that differs from Varnish's behavior but provides better separation of concerns:

### Phase 1: Parsing (pkg/parser)
- Parser recognizes `include "file.vcl";` statements
- Creates `IncludeDecl` AST nodes without resolving file contents
- Parser remains pure (no file I/O) and focused on syntax

### Phase 2: Resolution (pkg/include)
- `include.ResolveFile()` or `include.ResolveProgram()` processes `IncludeDecl` nodes
- Recursively reads and parses included files
- Merges all declarations into a single AST
- Handles errors: circular dependencies, missing files, syntax errors, depth limits

### Usage Patterns

**Simple API (most common):**
```go
// Parse main.vcl and all its includes
program, err := include.ResolveFile("main.vcl")
```

**Advanced API with options:**
```go
resolver := include.NewResolver(
    include.WithBasePath("/etc/varnish"),
    include.WithMaxDepth(5),
)
program, err := resolver.ResolveFile("main.vcl")
```

**Post-parsing resolution:**
```go
// Parse first, resolve later
program, err := parser.Parse(content, "main.vcl")
resolved, err := include.ResolveProgram(program)
```

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