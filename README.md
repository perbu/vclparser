# VCL Parser for Go

A VCL (Varnish Configuration Language) parser implemented in Go that parses VCL files into Abstract Syntax Trees (AST).

## Features

- Complete lexical analysis of VCL syntax
- Recursive descent parser with error recovery
- Type-safe AST representation
- Post-parse resolution of include statements
- Symbol table and semantic analysis
- Visitor pattern for AST traversal
- Support for VCL 4.0 and 4.1 syntax

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/varnish/vclparser/pkg/parser"
)

func main() {
    vclCode := `
    vcl 4.0;

    backend default {
        .host = "127.0.0.1";
        .port = "8080";
    }

    sub vcl_recv {
        if (req.method == "GET") {
            return (hash);
        }
    }
    `

    ast, err := parser.Parse(vclCode)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Parsed VCL with %d declarations\n", len(ast.Declarations))
}
```

## Architecture

- `pkg/lexer/` - Lexical analysis and tokenization
- `pkg/ast/` - AST node definitions and visitor pattern
- `pkg/parser/` - Recursive descent parser implementation
- `pkg/types/` - Type system and symbol table
- `examples/` - Usage examples
- `tests/testdata/` - Test VCL files

## VCL Language Support

This parser supports the full VCL language including:

- Version declarations (`vcl 4.0;`)
- Backend definitions with properties
- Access Control Lists (ACLs)
- Probe definitions
- Subroutine definitions
- All VCL statements (if/else, set, unset, call, return, etc.)
- Expression parsing with proper operator precedence
- Built-in variables and functions
- C-code blocks (C{ }C)

## Testing

```bash
go test ./...
```
