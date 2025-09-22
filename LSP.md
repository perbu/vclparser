# VCL Language Server Protocol (LSP) Implementation Guide

This document provides a comprehensive plan for implementing a Language Server Protocol (LSP) server for VCL (Varnish
Configuration Language) using the existing vclparser infrastructure.

## Architecture Overview

The VCL LSP server will be built as a standalone Go application that leverages the existing parser, AST, analyzer, and
VMOD infrastructure. The server will provide rich language features for VCL files through the LSP protocol.

## Core LSP Features Implementation

### 1. Text Document Synchronization

Implementation: Full document synchronization with incremental parsing optimization.

Components:

- `DocumentManager`: Track open documents, versions, and content
- `ChangeProcessor`: Handle document change events efficiently
- `ParseScheduler`: Debounce parsing operations for performance

Code Structure:

```go
package lsp

import (
	"sync"
	"time"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/parser"
)

type DocumentManager struct {
	documents map[string]*Document
	parser    *parser.Parser
	mutex     sync.RWMutex
}

type Document struct {
	URI        string
	Version    int
	Content    string
	AST        *ast.Program
	Errors     []analyzer.Error
	LastParsed time.Time
}

```

### 2. Diagnostics (Real-time Error Reporting)

Leverage: Existing `pkg/analyzer` with all validators (VMOD, return actions, variables, versions)

Implementation:

- Parse errors from `parser.DetailedError`
- Semantic errors from `analyzer.Analyzer.Analyze()`
- Include resolution errors from `pkg/include`
- VMOD validation errors from `VMODValidator`

Features:

- Syntax errors with rich context visualization
- Type mismatches in VMOD function calls
- Invalid return statements for VCL methods
- Undefined variable access
- VCL version compatibility issues
- Circular include dependencies

### 3. Completion (IntelliSense)

Context-Aware Completion Types:

1. VCL Keywords: `vcl`, `backend`, `sub`, `if`, `else`, `set`, `unset`, `call`, `return`
2. Built-in Variables: `req.method`, `beresp.status`, `client.ip`, etc. (from metadata)
3. VMOD Functions: All functions from loaded VMOD registry
4. User-Defined Symbols: Backends, subroutines, ACLs, probes
5. Return Actions: Context-specific returns (`lookup`, `pass`, `hit`, etc.)
6. Operators: VCL-specific operators (`~`, `!~`, `&&`, `||`)

Implementation Strategy:

```go
package lsp

import (
	"github.com/perbu/vclparser/pkg/metadata"
	"github.com/perbu/vclparser/pkg/vmod"
)

type CompletionProvider struct {
	symbolIndex  *SymbolIndex
	vmodRegistry *vmod.Registry
	metadata     *metadata.VCLMetadata
}

func (cp *CompletionProvider) ProvideCompletion(doc *Document, pos Position) []CompletionItem {
	// 1. Determine completion context (expression, statement, declaration)
	// 2. Get scope at position
	// 3. Filter relevant symbols based on context
	// 4. Generate completion items with documentation
}

```

### 4. Hover Information

Information Sources:

- VMOD function signatures and documentation
- Built-in VCL variable types and descriptions
- User-defined symbol information
- Type information for expressions

Implementation:

```go
package lsp

import "github.com/perbu/vclparser/pkg/metadata"

type HoverProvider struct {
	symbolIndex *SymbolIndex
	typeChecker *TypeChecker
	metadata    *metadata.VCLMetadata
}

func (hp *HoverProvider) ProvideHover(doc *Document, pos Position) *HoverInfo {
	// 1. Find AST node at position
	// 2. Determine symbol or expression type
	// 3. Generate rich markdown documentation
}

```

### 5. Go to Definition

Definition Types:

- Backend declarations
- Subroutine definitions
- ACL declarations
- Probe definitions
- VMOD imports
- Include file resolution

Implementation:

- Use existing symbol table from `pkg/types`
- Cross-reference tracking through AST visitor
- Multi-file definition support via include resolution

### 6. Find References

Reference Types:

- Variable usage across files
- Subroutine calls
- Backend references
- ACL usage
- VMOD function calls

Implementation:

```go
package lsp

type ReferenceProvider struct {
	symbolIndex *SymbolIndex
	workspace   *Workspace
}

func (rp *ReferenceProvider) FindReferences(uri string, pos Position) []Location {
	// 1. Identify symbol at position
	// 2. Search all workspace files for references
	// 3. Return locations with context
}

```

### 7. Document Symbols (Outline View)

Symbol Categories:

- VCL version declaration
- Import/include statements
- Backend definitions
- Subroutine definitions
- ACL definitions
- Probe definitions

Hierarchical Structure:

```go
package lsp

type DocumentSymbol struct {
	Name     string
	Kind     SymbolKind
	Range    Range
	Children []DocumentSymbol
}

```

### 8. Workspace Symbols (Global Search)

Implementation:

- Index all symbols across workspace
- Support fuzzy matching
- Categorize by symbol type
- Show file location and context

### 9. Rename (Safe Refactoring)

Rename Capabilities:

- Subroutine names (with call site updates)
- Backend names (with reference updates)
- ACL names
- Variable names (local scope)

Safety Checks:

- Validate new name doesn't conflict
- Check scope boundaries
- Preview changes before applying

### 10. Code Actions (Quick Fixes)

Quick Fix Types:

1. Import Missing VMOD: Auto-add import statements
2. Fix Return Statement: Suggest valid return actions
3. Add Missing Semicolon: Fix syntax errors
4. Convert String to Regex: For pattern matching
5. Extract Backend: Create backend from inline definition
6. Add Type Cast: Fix type mismatches

Refactoring Actions:

1. Extract Subroutine: Create subroutine from code block
2. Inline Variable: Replace variable with its value
3. Split Complex Condition: Break down complex if statements

### 11. Formatting

VCL Code Formatter Features:

- Consistent indentation (configurable: tabs vs spaces)
- Proper spacing around operators
- Alignment of property assignments
- Consistent brace placement
- Line wrapping for long expressions

Implementation:

```go
package lsp

type Formatter struct {
	config FormatConfig
}

type FormatConfig struct {
	IndentSize      int
	UseTabs         bool
	MaxLineLength   int
	AlignProperties bool
}

```

### 12. Semantic Tokens (Enhanced Syntax Highlighting)

Token Classifications:

- Keywords (`vcl`, `backend`, `sub`, etc.)
- Built-in variables (`req.method`, `beresp.status`)
- VMOD functions and objects
- User-defined identifiers
- String literals and regex patterns
- Numeric literals and durations
- Comments and documentation
- Operators and punctuation

### 13. Signature Help (Parameter Hints)

Function Signature Sources:

- VMOD function signatures from VCC files
- Built-in VCL function signatures
- Parameter documentation and types

Implementation:

```go
package lsp

import (
   "github.com/perbu/vclparser/pkg/metadata"
   "github.com/perbu/vclparser/pkg/vmod"
)

type SignatureHelpProvider struct {
   vmodRegistry *vmod.Registry
   metadata     *metadata.VCLMetadata
}

func (shp *SignatureHelpProvider) ProvideSignatureHelp(doc *Document, pos Position) *SignatureHelp {
   // 1. Find active function call at position
   // 2. Determine current parameter index
   // 3. Return signature with parameter highlighting
}

```

### 14. Code Lens (Inline Annotations)

Code Lens Types:

1. Subroutine References: Show reference count
2. Backend Health: Show backend status (if monitoring available)
3. Performance Metrics: Show call frequency (if profiling available)
4. Test Coverage: Show which subroutines have tests

### 15. Folding Ranges (Code Folding)

Foldable Regions:

- Subroutine bodies
- Backend definitions
- ACL entries
- Probe definitions
- Comment blocks
- Include groups

## Implementation Components

### Server Infrastructure

#### 1. LSP Server Framework

```go
package lsp

type Server struct {
	client     jsonrpc2.Conn
	workspace  *Workspace
	docManager *DocumentManager
	features   *FeatureProviders
	config     ServerConfig
}

type ServerConfig struct {
	LogLevel       string
	CacheSize      int
	ParseTimeout   time.Duration
	MaxFileSize    int64
	EnableFeatures []string
}
```

#### 2. Document Management

```go
package lsp

import (
	"sync"

	"github.com/perbu/vclparser/pkg/ast"
)

type DocumentManager struct {
	docs        map[string]*Document
	changesChan chan DocumentChange
	parser      *ParserService
	analyzer    *AnalyzerService
	mutex       sync.RWMutex
}

type Document struct {
	URI         string
	Version     int
	Content     string
	AST         *ast.Program
	Symbols     *SymbolTable
	Diagnostics []Diagnostic
	Metadata    DocumentMetadata
}

type DocumentChange struct {
	URI     string
	Version int
	Changes []TextDocumentContentChangeEvent
}

```

#### 3. Position Mapping Utilities

```go
package lsp

type PositionMapper struct {
	content   string
	lines     []int // byte offset of each line start
	lineCount int
}

func (pm *PositionMapper) OffsetToPosition(offset int) Position
func (pm *PositionMapper) PositionToOffset(pos Position) int
func (pm *PositionMapper) RangeToOffsets(rng Range) (start, end int)

```

#### 4. Incremental Parsing Strategy

```go
package lsp

import (
	"time"

	"github.com/perbu/vclparser/pkg/vmod"
)

type ParserService struct {
	cache    *ParserCache
	registry *vmod.Registry
	config   ParseConfig
}

type ParseConfig struct {
	MaxCacheSize      int
	TTL               time.Duration
	EnableIncremental bool
}

func (ps *ParserService) ParseDocument(uri string, content string) (*ParseResult, error)
func (ps *ParserService) ParseIncremental(uri string, changes []Change) (*ParseResult, error)

```

#### 5. Symbol Indexing

```go
package lsp

type SymbolIndex struct {
	symbols     map[string][]Symbol
	references  map[string][]Reference
	definitions map[string]Definition
	workspace   *Workspace
}

type Symbol struct {
	Name          string
	Kind          SymbolKind
	Location      Location
	Signature     string
	Documentation string
}

type Reference struct {
	Location Location
	Context  ReferenceContext
}

```

### Feature Providers

```go
package lsp

type FeatureProviders struct {
	Completion     *CompletionProvider
	Hover          *HoverProvider
	Definition     *DefinitionProvider
	References     *ReferenceProvider
	Symbols        *SymbolProvider
	Rename         *RenameProvider
	CodeActions    *CodeActionProvider
	Formatting     *FormattingProvider
	SemanticTokens *SemanticTokenProvider
	SignatureHelp  *SignatureHelpProvider
	CodeLens       *CodeLensProvider
	FoldingRanges  *FoldingRangeProvider
}

```

## Integration with Existing Infrastructure

### Leveraging Existing Packages

1. pkg/parser: Core parsing functionality
    - Use `parser.Parse()` for full document parsing
    - Leverage `parser.DetailedError` for rich diagnostics
    - Utilize token position information

2. pkg/ast: AST traversal and manipulation
    - Use visitor pattern for AST analysis
    - Leverage position information in nodes
    - Utilize existing node types and interfaces

3. pkg/analyzer: Semantic analysis
    - Use all existing validators
    - Leverage symbol table functionality
    - Utilize type checking capabilities

4. pkg/vmod: VMOD support
    - Use registry for completion and validation
    - Leverage function signature information
    - Utilize type information for parameters

5. pkg/metadata: VCL built-in knowledge
    - Use for built-in variable completion
    - Leverage method context information
    - Utilize type definitions

6. pkg/include: Multi-file support
    - Use for workspace-wide analysis
    - Leverage circular dependency detection
    - Utilize resolved AST functionality

### Cache and Performance Strategy

#### 1. Multi-Level Caching

```go
package lsp

import (
   "time"

   "github.com/perbu/vclparser/pkg/ast"
)

type CacheManager struct {
   parseCache      *LRUCache[string, *ast.Program]
   symbolCache     *LRUCache[string, *SymbolTable]
   diagnosticCache *LRUCache[string, []Diagnostic]
   ttl             time.Duration
}

```

#### 2. Incremental Updates

- Parse only changed regions when possible
- Invalidate affected caches smartly
- Background processing for non-blocking updates
- Debounce rapid changes

#### 3. Memory Management

- LRU eviction for parsed ASTs
- Configurable cache sizes
- Memory pressure monitoring
- Lazy loading of symbol information

## Development Roadmap

### Phase 1: Foundation (4-6 weeks)

Goal: Basic LSP functionality

Features:

- [ ] LSP server infrastructure with JSON-RPC transport
- [ ] Document synchronization (open, change, close)
- [ ] Basic diagnostics (parse errors + semantic errors)
- [ ] Keyword completion
- [ ] Simple hover information
- [ ] Document symbols (outline view)

Deliverables:

- Working LSP server executable
- VS Code extension (basic)
- Integration tests

### Phase 2: Core Features (6-8 weeks)

Goal: Essential language features

Features:

- [ ] Advanced completion (context-aware, VMOD functions)
- [ ] Go to definition (cross-file support)
- [ ] Find references
- [ ] Workspace symbols
- [ ] Enhanced hover with VMOD documentation
- [ ] Signature help for VMOD functions

Deliverables:

- Feature-complete core functionality
- Performance benchmarks
- Documentation

### Phase 3: Advanced Features (4-6 weeks)

Goal: Developer productivity features

Features:

- [ ] Rename refactoring
- [ ] Code actions and quick fixes
- [ ] Code formatting
- [ ] Semantic tokens (enhanced highlighting)
- [ ] Code lens
- [ ] Folding ranges

Deliverables:

- Advanced IDE integration
- Refactoring capabilities
- Code quality tools

### Phase 4: Optimization & Polish (2-4 weeks)

Goal: Production readiness

Features:

- [ ] Performance optimization
- [ ] Advanced caching strategies
- [ ] Configuration options
- [ ] Error recovery improvements
- [ ] Memory usage optimization

Deliverables:

- Performance-optimized server
- Configuration documentation
- Production deployment guide

## Testing Strategy

### Unit Tests

- Parser service tests
- Feature provider tests
- Position mapping tests
- Cache functionality tests

### Integration Tests

- End-to-end LSP protocol tests
- Multi-file workspace tests
- VMOD integration tests
- Include resolution tests

### Performance Tests

- Large file parsing benchmarks
- Memory usage profiling
- Concurrent request handling
- Cache efficiency measurements

### Real-World Tests

- Complex VCL configuration testing
- VS Code extension testing
- Multiple editor compatibility

## Deployment and Distribution

### Server Distribution

- Standalone Go binary
- Docker container
- Package manager distribution (brew, apt, etc.)

### Editor Extensions

- VS Code extension (primary target)
- Vim/Neovim LSP client configuration
- Emacs LSP client configuration
- IntelliJ plugin (future consideration)

### Configuration

- Server settings (cache size, timeouts, etc.)
- Client-specific configuration
- Workspace-specific settings
- VMOD path configuration

## Success Metrics

### Developer Experience

- Completion response time < 50ms
- Diagnostics update < 200ms after change
- Support for files up to 10MB
- Memory usage < 500MB for typical workspace

### Feature Completeness

- All major LSP features implemented
- VMOD integration working
- Multi-file analysis functional
- Real-time error detection accurate

### Adoption

- VS Code extension published
- Community feedback integration
- Documentation completeness
- Performance benchmarks published

## Future Enhancements

### Advanced Features

- VCL testing framework integration
- Performance profiling integration
- Varnish deployment integration
- Configuration validation
- Backend health monitoring
- Cache hit ratio analysis

### AI/ML Integration

- Code suggestions based on patterns
- Performance optimization recommendations
- Security vulnerability detection
- Best practice enforcement

### Tooling Integration

- Git integration for change tracking
- CI/CD pipeline integration
- Documentation generation
- Code review automation

This comprehensive plan leverages the excellent foundation provided by the existing vclparser infrastructure while
adding the necessary components for a production-ready LSP implementation. The phased approach ensures steady progress
while delivering value at each milestone.