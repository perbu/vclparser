# VCL Parser TODO and Future Enhancements

Parses VCL files into an AST. Can be given VCC files to validate VMOD usage.

## Missing Features

### Return Statement Action Keywords
- Parser requires parentheses around return actions, but VCL allows bare keywords
- Example: `return lookup;` fails (should work alongside `return (lookup);`)
- Keywords: `lookup`, `hash`, `pass`, `pipe`, `purge`, `synth`, `deliver`, `restart`

## Future Enhancements

### Parser Improvements
- Enhanced object literal parsing for complex backend/probe definitions
- Enhanced error recovery and partial parsing
- Advanced semantic analysis (variable scope, dead code detection, flow validation)

### Developer Experience
- Language Server Protocol (LSP) implementation
- VCL formatter
- IDE integration and VS Code extension
- Testing tools (fuzzing, performance benchmarks, compliance testing)

### Advanced Features
- ESI support
- String templates and advanced interpolation
- Time/duration arithmetic
- Regex validation

## VMOD Support

- 64/64 VMODs successfully loaded from vcclib directory (100% success rate)
- 605 functions and 37 objects available for validation
- Type-safe argument validation and error detection
- Support for all core VMODs: std, directors, crypto, ykey, vha, brotli, utils, etc.
- VCC file parsing and module registry system
- Enhanced type inference for VMOD function return types
- Context-aware enum literal recognition

## Performance

- Parse 1MB VCL files in <100ms
- <10MB memory usage
- Support for files with 10,000+ lines
- Comprehensive test coverage with real-world VCL examples