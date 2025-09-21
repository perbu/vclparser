# VCL Compiler JSON Metadata

This document describes the JSON metadata format exported by `lib/libvcc/generate.py` when called with the `JSON` argument.

## Usage

```bash
cd lib/libvcc
python3 generate.py JSON > vcl_metadata.json
```

The output is a single JSON document containing all VCL compiler metadata needed for implementing out-of-tree VCL compilers.

## JSON Structure

The root JSON object contains five top-level keys:

```json
{
  "vcl_methods": { ... },
  "vcl_variables": { ... },
  "vcl_types": { ... },
  "vcl_tokens": { ... },
  "storage_variables": [ ... ]
}
```

## vcl_methods

Maps VCL method names to their execution context and allowed return actions.

**Structure:**
```json
{
  "method_name": {
    "context": "C|B|H",
    "allowed_returns": ["action1", "action2", ...]
  }
}
```

**Context values:**
- `"C"` - Client-side methods (vcl_recv, vcl_deliver, etc.)
- `"B"` - Backend-side methods (vcl_backend_fetch, vcl_backend_response, etc.)
- `"H"` - Housekeeping methods (vcl_init, vcl_fini)

**Example:**
```json
{
  "recv": {
    "context": "C",
    "allowed_returns": [
      "fail", "synth", "restart", "pass", "pipe",
      "hash", "purge", "vcl", "connect"
    ]
  },
  "backend_fetch": {
    "context": "B",
    "allowed_returns": ["fail", "fetch", "abandon", "error"]
  }
}
```

**Usage:** Use this to validate that `return` statements in VCL methods use appropriate actions. For example, `return pipe;` is only valid in methods that include `"pipe"` in their `allowed_returns` array.

## vcl_variables

Maps VCL variable names to their type information and access permissions.

**Structure:**
```json
{
  "variable_name": {
    "type": "TYPE_NAME",
    "readable_from": ["method1", "method2", ...],
    "writable_from": ["method1", "method2", ...],
    "unsetable_from": ["method1", "method2", ...],
    "version_low": 0,
    "version_high": 99
  }
}
```

**Field descriptions:**
- `type` - VCL type name (see vcl_types section)
- `readable_from` - VCL method contexts where variable can be read
- `writable_from` - VCL method contexts where variable can be modified with `set`
- `unsetable_from` - VCL method contexts where variable can be removed with `unset`
- `version_low`/`version_high` - VCL version range where variable is available

**Context values in permission arrays:**
- `"all"` - Available in all methods
- `"client"` - Available in all client-side methods (context "C")
- `"backend"` - Available in all backend-side methods (context "B")
- `"both"` - Available in both client and backend methods
- Specific method names like `"recv"`, `"deliver"`, etc.

**Examples:**
```json
{
  "client.ip": {
    "type": "IP",
    "readable_from": ["client", "backend"],
    "writable_from": [],
    "unsetable_from": [],
    "version_low": 0,
    "version_high": 99
  },
  "req.url": {
    "type": "STRING",
    "readable_from": ["recv", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"],
    "writable_from": ["recv"],
    "unsetable_from": [],
    "version_low": 0,
    "version_high": 99
  },
  "req.http.cookie": {
    "type": "HEADER",
    "readable_from": ["recv", "pass", "hash", "purge", "miss", "hit", "deliver", "synth"],
    "writable_from": ["recv"],
    "unsetable_from": ["recv"],
    "version_low": 0,
    "version_high": 99
  }
}
```

**Usage:**
- Validate variable access: check that variables are only read/written in permitted contexts
- Type checking: ensure assignments match the variable's type
- Feature availability: check version_low/version_high for VCL version compatibility

## vcl_types

Defines the VCL type system and C type mappings.

**Structure:**
```json
{
  "TYPE_NAME": {
    "c_type": "c_type_string",
    "internal": boolean
  }
}
```

**Field descriptions:**
- `c_type` - Corresponding C type in generated code
- `internal` - Whether this is an internal type not directly accessible in VCL

**Examples:**
```json
{
  "STRING": {
    "c_type": "const char *",
    "internal": false
  },
  "IP": {
    "c_type": "VCL_IP",
    "internal": false
  },
  "BOOL": {
    "c_type": "unsigned",
    "internal": false
  },
  "HEADER": {
    "c_type": "VCL_HEADER",
    "internal": false
  },
  "STRINGS": {
    "c_type": "void",
    "internal": true
  }
}
```

**Usage:**
- Type validation in VCL expressions
- Code generation for C backend
- Understanding type compatibility rules

## vcl_tokens

Defines lexical tokens used by the VCL parser.

**Structure:**
```json
{
  "TOKEN_NAME": "token_string"
}
```

**Examples:**
```json
{
  "T_INC": "++",
  "T_DEC": "--",
  "T_CAND": "&&",
  "T_COR": "||",
  "T_LEQ": "<=",
  "T_EQ": "==",
  "T_NEQ": "!=",
  "T_GEQ": ">=",
  "'+'": "+",
  "'('": "(",
  "')'": ")"
}
```

**Usage:**
- Lexer implementation
- Syntax highlighting
- Parser development

## storage_variables

Defines storage-engine specific variables (like `storage.<name>.free_space`).

**Structure:**
```json
[
  {
    "name": "variable_name",
    "type": "TYPE_NAME",
    "default": "default_value",
    "description": "short_description",
    "docstring": "detailed_documentation"
  }
]
```

**Examples:**
```json
[
  {
    "name": "free_space",
    "type": "BYTES",
    "default": "0.",
    "description": "storage.<name>.free_space",
    "docstring": "Free space available in the named stevedore. Only available for\nthe malloc stevedore."
  },
  {
    "name": "used_space",
    "type": "BYTES",
    "default": "0.",
    "description": "storage.<name>.used_space",
    "docstring": "Used space in the named stevedore. Only available for the malloc\nstevedore."
  }
]
```

**Usage:**
- Validate storage-specific variable access
- Generate documentation for storage variables
- Implement storage backend variable handling

## Implementation Notes

### Method Context Resolution

When a variable lists `"client"` or `"backend"` in its permission arrays, resolve these to specific methods using the `vcl_methods` context information:

```python
def resolve_context(permission, vcl_methods):
    if permission == "client":
        return [name for name, info in vcl_methods.items() if info["context"] == "C"]
    elif permission == "backend":
        return [name for name, info in vcl_methods.items() if info["context"] == "B"]
    elif permission == "both":
        return [name for name, info in vcl_methods.items() if info["context"] in ["C", "B"]]
    else:
        return [permission]  # Specific method name
```

### Header Variables

Variables with type `"HEADER"` represent HTTP headers and follow the pattern:
- `req.http.*` - Request headers (readable/writable in client methods)
- `bereq.http.*` - Backend request headers (readable/writable in backend methods)
- `resp.http.*` - Response headers (readable/writable in client methods)
- `beresp.http.*` - Backend response headers (readable/writable in backend methods)

### Version Compatibility

Use `version_low` and `version_high` to check if a variable is available in a specific VCL version:

```python
def is_variable_available(variable, vcl_version):
    return variable["version_low"] <= vcl_version <= variable["version_high"]
```

### Generating This Metadata

The JSON metadata is generated from the same sources used by the official VCL compiler:
- Method/action mappings from the `returns` tuple in `generate.py`
- Variable definitions parsed from `doc/sphinx/reference/vcl_var.rst`
- Type system from VRT headers and internal definitions
- Token definitions from the lexer specification

This ensures the metadata stays synchronized with the official Varnish VCL specification.