# Analyzer Package

The analyzer package provides *semantic* analysis capabilities for VCL Abstract Syntax Trees. It performs comprehensive
validation beyond parsing to ensure VCL code correctness and compatibility.

## Features

- VMOD Validation: Validates VMOD imports, function calls, object instantiation, and named parameter syntax
- Return Action Validation: Ensures return statements use valid actions for their VCL method context
- Variable Access Validation: Validates variable read/write permissions against VCL metadata
- Version Compatibility: Checks variable and feature usage against VCL version constraints
- Symbol Table Management: Tracks modules, objects, backends, and functions for cross-referencing

## Validators

- VMODValidator: VMOD function calls, object methods, named parameters, restrictions
- ReturnActionValidator: Return statement actions in built-in VCL subroutines
- VariableAccessValidator: Variable read/write/unset permissions by method context
- VersionValidator: VCL version compatibility for variables and features

## Integration

The analyzer integrates with the parser package to provide complete VCL processing and works with the metadata package
for VCL language definitions and the vmod package for VMOD registry management.