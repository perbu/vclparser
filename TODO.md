# VCL Parser TODO and Future Enhancements

Parses VCL files into an AST. Can be given VCC files to validate VMOD usage.

## Missing Features

### Known Edge Cases (VMOD Type Inference)

#### Named Argument Mapping Issue
- **Issue**: `utils.time_format("%format", time = std.real2time(-1, now))` causes format string to be mapped to wrong parameter position
- **Error**: "argument 2: expected BOOL, got STRING"
- **Root Cause**: Named argument mapping logic in `buildCompleteArgumentList` needs refinement
- **Files**: `analyzer/vmod_validator.go:228-295`

## Future Enhancements

### Potential Parser Improvements
- Enhanced object literal parsing for complex backend/probe definitions
- Enhanced error recovery and partial parsing

### Developer Experience
- Language Server Protocol (LSP) implementation
- VCL formatter
- IDE integration and VS Code extension
- Testing tools (fuzzing, performance benchmarks, compliance testing)

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