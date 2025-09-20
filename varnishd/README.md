# Varnishd Code Generation Tools

This directory contains code generation tools lifted from the Varnish HTTP accelerator project.

## generate.py

A Python script that generates various C header files and source files for the VCL (Varnish Configuration Language) compiler.

### Main Purpose
- Generates C code definitions for the VCL language lexer, parser, and type system
- Creates header files that define tokens, VCL methods, return values, and variable access patterns

### Key Outputs
1. **Token definitions** (`vcc_token_defs.h`) - Defines lexical tokens like `T_INC` for `++`, `T_EQ` for `==`, etc.
2. **VCL method/return mappings** (`vcl_returns.h`) - Defines which return actions are valid in which VCL methods (recv, hit, miss, etc.)
3. **Variable access control** (`vcc_obj.c`) - Generates code that controls which VCL variables can be read/written in which contexts
4. **Type system** - Creates C representations of VCL types (STRING, INT, BOOL, etc.)

### How it works
- Parses VCL variable documentation to extract read/write permissions
- Generates C code that enforces these permissions at compile time
- Creates token recognition functions for the lexer
- Maps VCL methods to their allowed return actions (e.g., `vcl_recv` can return `pass`, `hash`, `pipe`, etc.)

This is the build system that creates the foundation for Varnish's VCL compiler - it automates generating all the boilerplate C code needed to implement the VCL language parser and type checker.

## vmodtool.py

A Python script that generates C interfaces and documentation for Varnish Modules (VMODs) from `.vcc` specification files.

### Purpose
- Reads VMOD specification files (`.vcc`) and generates C header files (`vmod_if.h`) with function prototypes
- Creates implementation glue code (`vmod_if.c`) that integrates the VMOD with Varnish
- Extracts documentation in reStructuredText format (`vmod_${name}.rst`)
- Provides build system integration through autotools boilerplate

This tool is essential for VMOD development in Varnish, automating the interface generation between VCL and C code.