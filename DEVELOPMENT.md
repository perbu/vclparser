# DEVELOPMENT.md

## Package Architecture

The VCL parser is organized into distinct packages with clear separation of concerns:

```
pkg
├── lexer/       # Tokenization
├── parser/      # Recursive descent parsing
├── ast/         # Abstract syntax tree definitions
├── types/       # Type system and symbol tables
├── analyzer/    # Semantic analysis
├── vmod/        # VMOD registry and management
├── vcc/         # VCC file parsing (VMOD definitions)
├── metadata/    # VCL compiler metadata and validation
```

## Core Parsing Pipeline

### pkg/lexer/
Purpose: Tokenizes VCL source code into lexical tokens
- `lexer.go`: Main lexer implementation with position tracking
- `token.go`: Token definitions and types
- `lexer_test.go`: Lexer unit tests

The lexer performs character-by-character scanning with lookahead support. Tracks line/column positions for error reporting.

### pkg/parser/
Purpose: Recursive descent parser that converts tokens to AST
- `parser.go`: Main parser entry point and infrastructure
- `expressions.go`: Expression parsing with operator precedence
- `statements.go`: Statement parsing (if/else, assignments, calls)
- `declarations.go`: Top-level declaration parsing (backends, subroutines)
- `duration.go`: VCL duration literal parsing
- `error.go`: Parser error handling and recovery
- `named_arguments_test.go`: Tests for VMOD named parameter syntax
- `*_test.go`: Comprehensive parsing tests

Parser follows grammar productions closely. Implements error recovery to continue parsing after syntax errors.

### pkg/ast/
Purpose: AST node definitions and visitor pattern implementation
- `node.go`: Base AST node interfaces and common types
- `expressions.go`: Expression AST nodes (binary ops, calls, literals)
- `statements.go`: Statement AST nodes (if, assignments, returns)
- `visitor.go`: Visitor pattern for AST traversal

All nodes implement position tracking for source mapping. Visitor pattern enables multiple analysis passes.

### pkg/types/
Purpose: Type system and symbol table management
- `types.go`: VCL type definitions (STRING, INT, BACKEND, etc.)
- `symbol_table.go`: Scoped symbol table for variables and functions

Implements VCL's type system including built-in types and type checking rules.

## Extended Functionality

### pkg/analyzer/
Purpose: Semantic analysis on parsed AST
- `analyzer.go`: Main semantic analysis coordinator
- `vmod_validator.go`: VMOD usage validation and type checking
- `vmod_validator_test.go`: VMOD validation tests

Validates VMOD function calls, parameter types, and usage patterns. Extensible for additional semantic checks.

### pkg/vmod/
Purpose: VMOD registry and definition management
- `registry.go`: VMOD definition loading and lookup
- `registry_test.go`: Registry functionality tests
- `*_test.go`: Integration tests with real VMOD definitions

Loads VMOD definitions from VCC files and provides runtime lookup for validation.

### pkg/vcc/
Purpose: VCC file parsing for VMOD definitions
- `parser.go`: VCC file parser
- `types.go`: VCC-specific types and structures
- `lexer.go`: VCC tokenizer
- `lexer_simple.go`: Simplified lexer implementation
- `*_test.go`: VCC parsing tests

Parses Varnish VCC (Varnish C Compiler) files that define VMOD interfaces and function signatures.

### pkg/metadata/
Purpose: VCL compiler metadata for semantic validation
- `types.go`: Type definitions for VCL metadata structures
- `loader.go`: Embedded metadata loading and validation APIs
- `metadata.json`: JSON metadata exported from varnishd's generate.py
- `README.md`: Documentation of metadata format and usage

Provides embedded VCL metadata from the official Varnish compiler including:
- VCL methods with allowed return actions
- VCL variables with type information and access permissions
- VCL type system definitions
- Lexical tokens
- Storage-engine specific variables

## Data Flow

1. Tokenization: `lexer` converts VCL source to token stream
2. Parsing: `parser` builds AST from tokens using `ast` node types
3. Type Resolution: `types` provides type checking infrastructure
4. VMOD Loading: `vmod` registry loads definitions via `vcc` parser
5. Metadata Loading: `metadata` provides embedded VCL compiler metadata
6. Analysis: `analyzer` performs semantic validation using symbol tables, VMOD registry, and VCL metadata

## Integration Points

- parser → ast: Parser creates AST nodes
- parser → lexer: Parser consumes tokens from lexer
- parser → vmod: Parser validates VMOD calls against registry
- analyzer → vmod: Analyzer uses registry for semantic validation
- analyzer → types: Analyzer uses type system for validation
- analyzer → metadata: Analyzer uses VCL metadata for variable/method validation
- vmod → vcc: Registry loads VMOD definitions via VCC parser
- metadata → embedded: Metadata loads from embedded JSON data at compile time

## Extension Points

- ast/visitor.go: Add new analysis passes by implementing Visitor interface
- analyzer/: Add semantic checks by extending analyzer
- types/: Extend type system for custom types
- vmod/: Add VMOD loading from other sources beyond VCC files
- metadata/: Update embedded metadata when varnishd definitions change

## Testing Structure

- Unit tests in each package test individual components
- `tests/` contains integration tests exercising full parsing pipeline
- Test data in `tests/testdata/` provides real VCL examples
- VMOD tests use fixtures from `vcclib/` directory