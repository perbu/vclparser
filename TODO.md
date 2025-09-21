# VCL Parser TODO and Future Enhancements

Parses VCL files into an AST. Can be given VCC files to validate VMOD usage.

## Missing Features

### Parser Limitations

None.

## Missing Parser Features for LSP Support

### Core Parser Enhancements Needed

#### Comment Preservation in AST
- Issue: Comments are currently skipped during parsing (`parser.go:82-84`)
- Need: Preserve comments in AST for documentation hover and code formatting
- Implementation: Add comment nodes to AST, attach to relevant declarations
- Files: `pkg/parser/parser.go`, `pkg/ast/node.go`

#### Partial/Incremental Parsing
- Issue: Parser requires complete, valid VCL files
- Need: Parse incomplete/invalid code for real-time editing in IDEs
- Implementation: Add error recovery modes, partial AST construction
- Files: `pkg/parser/parser.go`, `pkg/parser/error.go`

#### Token-to-AST Node Mapping
- Issue: No bidirectional mapping between tokens and AST nodes
- Need: Efficient position-to-node lookup for LSP features (hover, completion)
- Implementation: Add token ranges to all AST nodes, position index
- Files: `pkg/ast/node.go`, `pkg/parser/parser.go`

#### Documentation Comment Parsing
- Issue: No support for documentation comments (/ */, ///)
- Need: Parse and attach documentation to declarations for hover info
- Implementation: Recognize doc comment patterns, associate with following declarations
- Files: `pkg/lexer/lexer.go`, `pkg/ast/node.go`

#### Semantic Token Classification
- Issue: No token classification beyond basic lexing
- Need: Classify tokens for semantic syntax highlighting (keywords, variables, functions)
- Implementation: Add semantic analysis pass, token classification visitor
- Files: New `pkg/semantics/tokens.go`

#### Scope Analysis at Position
- Issue: No way to determine scope context at arbitrary positions
- Need: LSP completion needs to know what symbols are available at cursor
- Implementation: Position-aware scope resolution, context detection
- Files: `pkg/types/symbol_table.go`, new `pkg/scope/analyzer.go`

#### Cross-Reference Index
- Issue: No efficient way to find all references to a symbol
- Need: "Find References" and "Rename" LSP features
- Implementation: Build reference index during parsing, track symbol usage
- Files: New `pkg/index/references.go`

#### Type Inference Enhancement
- Issue: Limited type inference for expressions
- Need: Complete type information for hover and completion
- Implementation: Enhance type checker, infer types for all expressions
- Files: `pkg/types/types.go`, `pkg/analyzer/analyzer.go`

#### AST Diff/Merge Support
- Issue: No support for incremental AST updates
- Need: Efficient updates for document changes in LSP
- Implementation: AST node diffing, selective re-parsing
- Files: New `pkg/ast/diff.go`

#### Whitespace and Formatting Preservation
- Issue: AST doesn't preserve formatting information
- Need: Code formatter needs original formatting context
- Implementation: Store whitespace info in AST, formatting-aware parser
- Files: `pkg/ast/node.go`, new `pkg/format/preserving_parser.go`

#### Call Graph Construction
- Issue: No analysis of subroutine call relationships
- Need: LSP code lens, call hierarchy features
- Implementation: Build call graph during analysis, track subroutine relationships
- Files: New `pkg/analysis/callgraph.go`

#### Completion Context Detection
- Issue: No way to determine what completions are valid at a position
- Need: Context-aware completion (expressions vs statements vs declarations)
- Implementation: Context analysis visitor, completion scope detection
- Files: New `pkg/completion/context.go`

#### Error Recovery Improvements
- Issue: Parser stops on first error, doesn't recover well
- Need: Continue parsing after errors for better IDE experience
- Implementation: Enhanced error recovery strategies, partial AST construction
- Files: `pkg/parser/parser.go`, `pkg/parser/error.go`

#### Dependency Graph for Includes
- Issue: No efficient tracking of include dependencies
- Need: LSP workspace analysis, change propagation
- Implementation: Build dependency graph, change impact analysis
- Files: `pkg/include/dependencies.go`

## Future Enhancements

### Potential downstream products
- VCL compiler, naturally
- VCL formatter with whitespace preservation
- LSP

## Performance

- Parse 1MB VCL files in <100ms
- <10MB memory usage
- Support for files with 10,000+ lines
- Comprehensive test coverage with real-world VCL examples