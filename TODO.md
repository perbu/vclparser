# VCL Parser TODO and Future Enhancements

Parses VCL files into an AST. Can be given VCC files to validate VMOD usage.

## Recently Completed Features (2025)

### ✅ Return Statement Action Keywords (Fixed)
- Parser now supports both parenthesized and naked return statements
- Examples: Both `return lookup;` and `return (lookup);` work correctly
- All valid VCL return action keywords supported: `lookup`, `hash`, `pass`, `pipe`, `purge`, `synth`, `deliver`, `restart`, `fetch`, `miss`, `hit`, `abandon`, `retry`, `error`, `ok`, `fail`, `vcl`
- Comprehensive test coverage added in `parser/return_actions_test.go`
- Full backward compatibility maintained

## Missing Features

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