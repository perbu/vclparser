# VCL Parser TODO and Future Enhancements

This document outlines planned improvements and extensions for the VCL parser implementation.

## Current Status

The VCL parser now has comprehensive VMOD support including:
- ✅ VMOD object instantiation (`new` keyword)
- ✅ VMOD function call validation
- ✅ Import validation and type checking
- ✅ VCC file parsing and registry system
- ✅ Comprehensive test coverage for real-world VMOD usage patterns

## Current Limitations

### Named Parameter Syntax
- Issue: VCL parser doesn't support named parameter syntax from vmod-vcl.md examples
- Example: `s3.verify(access_key_id = "KEY", secret_key = "SECRET")` fails to parse
- Impact: Some real-world VCL examples require positional parameters instead
- Priority: Medium
- Workaround: Use positional parameters: `s3.verify("KEY", "SECRET", 1s)`

### Complex VCC Types
- Issue: Some advanced VCC types are not fully supported
- Example: `BEREQ` type in `headerplus.init(BEREQ bereq)` causes VCC parse errors
- Impact: Some VMODs from vcclib directory fail to load
- Priority: Medium
- Status: 40+ VMODs load successfully, but some with advanced types fail

### Duration Literal Type Inference
- Issue: Duration literals like `1s` sometimes inferred as STRING instead of DURATION
- Example: `s3.verify("key", "secret", 1s)` fails validation
- Impact: Minor type validation issues with duration parameters
- Priority: Low
- Workaround: Most duration usage works correctly

## Planned Enhancements

### 1. Enhanced Object Literal Support
- [ ] Nested Object Parsing: Support for complex backend/probe definitions
- [ ] Property Validation: Ensure only valid properties are used in each context
- [ ] Default Values: Handle optional properties with sensible defaults
- [ ] Type Checking: Validate property types (string, duration, integer, etc.)

Example:
```vcl
backend web {
    .host = "example.com";
    .probe = {
        .url = "/health";
        .interval = 30s;
        .timeout = 5s;
        .window = 5;
        .threshold = 3;
        .initial = 2;
    };
}
```

### 2. Advanced VCL Syntax Support
- [ ] Named Parameters: Support `func(param = value)` syntax
- [ ] Complex Expression Parsing: Ternary operators, complex string interpolation
- [ ] Advanced VCC Types: Full support for BEREQ, PRIV_TASK, etc.
- [ ] Enum Defaults: Support for enum parameters with default values

### 3. Enhanced Semantic Analysis
- [ ] Variable Scope Analysis: Track variable accessibility across VCL methods
- [ ] Method Flow Validation: Ensure proper VCL method call sequences
- [ ] Return Value Checking: Validate return statements match method requirements
- [ ] Dead Code Detection: Identify unreachable code after return statements
- [ ] Variable Usage Analysis: Detect unused variables and invalid access patterns

### 4. Code Generation and Transformation
- [ ] VCL Formatter: Pretty-print VCL with consistent style
- [ ] VCL Minifier: Remove comments and unnecessary whitespace
- [ ] Version Migration: Upgrade VCL from 4.0 to 4.1 automatically
- [ ] Configuration Optimization: Suggest performance improvements
- [ ] Documentation Generator: Create HTML/Markdown docs from VCL

### 5. Advanced Error Recovery
- [ ] Partial Parsing: Continue parsing after encountering errors
- [ ] Error Suggestions: Provide fix suggestions for common mistakes
- [ ] Multiple Error Reporting: Report all errors in a single pass
- [ ] Syntax Highlighting: Support for IDE integration with error markers

### 6. Language Server Protocol (LSP) Support
- [ ] Autocomplete: Variable and function name completion
- [ ] Hover Information: Show variable types and function signatures
- [ ] Go to Definition: Jump to variable/function definitions
- [ ] Diagnostics: Real-time error and warning reporting
- [ ] Refactoring: Rename variables across files

### 7. Testing and Quality Assurance
- [ ] Fuzzing: Automated testing with malformed VCL inputs
- [ ] Performance Benchmarks: Measure parsing speed with large VCL files
- [ ] Memory Profiling: Optimize memory usage for large ASTs
- [ ] Compliance Testing: Test against official VCL test suite
- [ ] Edge Case Coverage: Handle unusual but valid VCL constructs

### 8. Integration and Tooling
- [ ] CLI Tools: Command-line utilities for VCL validation and formatting
- [ ] GitHub Actions: CI/CD integration for VCL validation
- [ ] VS Code Extension: Syntax highlighting and error checking
- [ ] Web Interface: Online VCL validator and formatter
- [ ] API Server: RESTful API for VCL parsing and validation

### 9. Advanced VCL Features
- [ ] ESI Support: Parse and validate Edge Side Includes
- [ ] Regular Expression Validation: Check regex syntax in VCL
- [ ] Time/Duration Arithmetic: Validate time calculations
- [ ] IP Address Validation: Ensure valid IP addresses and CIDR ranges
- [ ] String Template Support: Advanced string interpolation

### 10. Documentation and Learning
- [ ] Interactive Tutorial: Web-based VCL learning tool
- [ ] Best Practices Guide: Automated VCL code review suggestions
- [ ] Migration Guide: Help transitioning from other cache solutions
- [ ] Performance Guide: VCL optimization recommendations
- [ ] Security Guide: Security best practices validation

## Implementation Roadmap

### Phase 1: Syntax Completeness (Next 2-3 months)
1. Named parameter syntax support
2. Enhanced object literal parsing for backend properties
3. Advanced VCC type support (BEREQ, PRIV_TASK, etc.)
4. Duration literal type inference improvements
5. VCL formatter implementation

### Phase 2: Advanced Analysis (Next 6 months)
1. Complete semantic analysis
2. LSP server implementation
3. Enhanced VMOD registry features
4. Code generation tools

### Phase 3: Ecosystem Integration (Next year)
1. IDE extensions and plugins
2. CI/CD integrations
3. Web-based tools
4. Advanced optimization features

## Contributing

Contributions are welcome! Current priorities:

1. **Named Parameter Parsing** - Most impactful missing syntax feature
2. **Advanced VCC Types** - Improve VMOD coverage by supporting complex types
3. **Object Literal Enhancement** - Better support for nested backend/probe definitions
4. **Test Coverage Expansion** - More edge cases and real-world VCL files
5. **Documentation** - Usage examples and API documentation

## Performance Goals

- Parse 1MB VCL files in under 100ms
- Memory usage under 10MB for typical VCL configurations
- Support for files with 10,000+ lines
- Incremental parsing for LSP responsiveness

## Compatibility

The parser maintains compatibility with:
- VCL 4.0 and VCL 4.1 syntax
- Varnish Cache 6.x and 7.x built-in variables
- Standard VMODs shipped with Varnish
- Enterprise VMODs from Varnish Software (40+ VMODs successfully loaded)

## VMOD Support Status

The parser now includes comprehensive VMOD support:

### ✅ Implemented Features
- VMOD import validation
- Function call type checking
- Object instantiation and method calls
- VCC file parsing and module registry
- Type-safe argument validation
- Error reporting for missing imports and type mismatches

### 📊 VMOD Coverage
- **40+ VMODs** successfully loaded from vcclib
- **Core VMODs** fully supported: std, directors, crypto, ykey, vha
- **Enterprise VMODs** partially supported (some fail due to advanced VCC syntax)

### 🧪 Test Coverage
- Comprehensive integration tests for real-world usage patterns
- Type validation tests demonstrating correct error detection
- VMOD registry functionality tests
- Examples based on actual production VCL from vmod-vcl.md

This parser now provides robust VMOD support suitable for production VCL validation and analysis.