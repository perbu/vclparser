package parser

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
	"github.com/perbu/vclparser/pkg/types"
)

// Config contains parser configuration options
type Config struct {
	// DisableInlineC disables parsing of C code blocks (C{ }C)
	DisableInlineC bool
	// MaxErrors limits the number of errors before stopping parsing (0 = no limit)
	MaxErrors int
	// AllowMissingVersion allows parsing VCL files without version declarations
	// This is useful for included files which don't need their own version declaration
	AllowMissingVersion bool
	// SkipSubroutineValidation skips validation of subroutine calls during parsing
	// This is useful for included files where subroutines may be defined in other files
	SkipSubroutineValidation bool
}

// DefaultConfig returns the default parser configuration
func DefaultConfig() *Config {
	return &Config{
		DisableInlineC: false,
		MaxErrors:      8, // Stop after 8 errors by default
	}
}

// Parser implements a recursive descent parser for VCL
type Parser struct {
	lexer       *lexer.Lexer
	errors      []DetailedError
	input       string // Store original VCL source for error context
	filename    string // Store filename for error reporting
	symbolTable *types.SymbolTable
	config      *Config      // Parser configuration
	program     *ast.Program // Current program being built (used for subroutine merging)

	currentToken lexer.Token
	peekToken    lexer.Token

	// Recovery state
	panicMode        bool // Are we currently in error recovery?
	synchronizing    bool // Are we synchronizing to a recovery point?
	maxErrorsReached bool // Have we reached the maximum error limit?

	// Comment handling
	leadingComments []ast.Comment // Comments collected before the next node
	lastLine        int           // Last line number of the previous token (for trailing comment detection)
}

// New creates a new parser with default configuration
func New(l *lexer.Lexer, input, filename string) *Parser {
	return NewWithConfig(l, input, filename, DefaultConfig())
}

// NewWithConfig creates a new parser with the specified configuration
func NewWithConfig(l *lexer.Lexer, input, filename string, config *Config) *Parser {
	if config == nil {
		config = DefaultConfig()
	}

	p := &Parser{
		lexer:       l,
		errors:      []DetailedError{},
		input:       input,
		filename:    filename,
		symbolTable: types.NewSymbolTable(),
		config:      config,
	}

	// Read two tokens, so currentToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// Parse parses the input and returns the AST using default configuration
func Parse(input, filename string) (*ast.Program, error) {
	return ParseWithConfig(input, filename, DefaultConfig())
}

// ParseWithConfig parses the input and returns the AST using the specified configuration
func ParseWithConfig(input, filename string, config *Config) (*ast.Program, error) {
	l := lexer.New(input, filename)
	p := NewWithConfig(l, input, filename, config)
	program := p.ParseProgram()

	if len(p.errors) > 0 {
		// Return the first error
		return program, p.errors[0]
	}

	return program, nil
}

// ParseWithVMODValidation parses VCL input and performs VMOD validation
func ParseWithVMODValidation(input, filename string) (*ast.Program, []string, error) {
	// Parse the VCL code
	program, err := Parse(input, filename)
	if err != nil {
		return program, nil, err
	}

	// VMOD registry is automatically initialized with embedded VCC files
	// via the package init() function, so no explicit loading needed here

	// Return the program and empty validation errors
	// The validation will be handled by the analyzer package
	return program, []string{}, nil
}

// Errors returns all parsing errors
func (p *Parser) Errors() []DetailedError {
	return p.errors
}

// nextToken advances to the next token and collects any comments
func (p *Parser) nextToken() {
	p.lastLine = p.currentToken.End.Line
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()

	// Collect comments instead of skipping them
	for p.peekToken.Type == lexer.COMMENT {
		comment := ast.Comment{
			Text:    p.peekToken.Value,
			Start:   p.peekToken.Start,
			End:     p.peekToken.End,
			IsBlock: len(p.peekToken.Value) >= 2 && p.peekToken.Value[0] == '/' && p.peekToken.Value[1] == '*',
		}
		p.leadingComments = append(p.leadingComments, comment)
		p.peekToken = p.lexer.NextToken()
	}
}

// consumeComments returns and clears the collected leading comments
func (p *Parser) consumeComments() []ast.Comment {
	comments := p.leadingComments
	p.leadingComments = nil
	return comments
}

// attachComments attaches collected comments to a node
func (p *Parser) attachComments(node ast.Node, leading []ast.Comment) {
	if len(leading) > 0 || p.hasTrailingComment() {
		nodeComments := &ast.NodeComments{
			Leading: leading,
		}
		node.SetComments(nodeComments)
	}
}

// hasTrailingComment checks if there's a potential trailing comment on the same line
func (p *Parser) hasTrailingComment() bool {
	// Check if we have leading comments and the first one is on the same line as the last token
	if len(p.leadingComments) > 0 {
		firstComment := p.leadingComments[0]
		// A trailing comment is on the same line as the previous token
		if firstComment.Start.Line == p.lastLine {
			return true
		}
	}
	return false
}

// extractTrailingComment removes and returns a trailing comment if present
func (p *Parser) extractTrailingComment() *ast.Comment {
	if len(p.leadingComments) > 0 {
		firstComment := p.leadingComments[0]
		// A trailing comment is on the same line as the previous token
		if firstComment.Start.Line == p.lastLine {
			trailing := firstComment
			p.leadingComments = p.leadingComments[1:]
			return &trailing
		}
	}
	return nil
}

// addError adds a parsing error
func (p *Parser) addError(message string) {
	p.errors = append(p.errors, DetailedError{
		Message:  message,
		Position: p.currentToken.Start,
		Token:    p.currentToken,
		Filename: p.filename,
		Source:   p.input,
	})
	if p.hasReachedMaxErrors() {
		p.maxErrorsReached = true
	}
}

// addPeekError adds a parsing error using the peek token's position
func (p *Parser) addPeekError(message string) {
	p.errors = append(p.errors, DetailedError{
		Message:  message,
		Position: p.peekToken.Start,
		Token:    p.peekToken,
		Filename: p.filename,
		Source:   p.input,
	})
	if p.hasReachedMaxErrors() {
		p.maxErrorsReached = true
	}
}

// reportError adds error and enters panic mode if not already synchronizing
func (p *Parser) reportError(message string) {
	p.addError(message)
	if !p.synchronizing {
		p.panicMode = true
	}
}

// synchronize exits panic mode when reaching a recovery point.
// Resets parser state to normal operation after error recovery,
// allowing parsing to continue from a stable syntactic position.
func (p *Parser) synchronize() {
	p.panicMode = false
	p.synchronizing = false
}

// hasReachedMaxErrors checks if the parser has reached the maximum error limit
func (p *Parser) hasReachedMaxErrors() bool {
	if p.config.MaxErrors == 0 {
		return false // 0 means unlimited
	}
	return len(p.errors) >= p.config.MaxErrors
}

// expectToken checks if current token matches expected type
func (p *Parser) expectToken(t lexer.TokenType) bool {
	if p.currentToken.Type == t {
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s", t, p.currentToken.Type))
	return false
}

// expectPeek checks if peek token matches expected type and advances
func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekToken.Type == t {
		p.nextToken()
		return true
	}
	p.addPeekError(fmt.Sprintf("expected next token to be %s, got %s", t, p.peekToken.Type))
	return false
}

// currentTokenIs checks if current token is of given type
func (p *Parser) currentTokenIs(t lexer.TokenType) bool {
	return p.currentToken.Type == t
}

// peekTokenIs checks if peek token is of given type
func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

// skipSemicolon optionally skips a semicolon
func (p *Parser) skipSemicolon() {
	if p.currentTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
}

// skipToSynchronizationPoint advances tokens until finding a recovery point.
// Used during error recovery to skip past malformed syntax until reaching
// a token that likely represents the start of a new syntactic construct
// (e.g., statement keywords, braces, semicolons).
func (p *Parser) skipToSynchronizationPoint(syncTokens ...lexer.TokenType) {
	for !p.currentTokenIs(lexer.EOF) {
		for _, token := range syncTokens {
			if p.currentTokenIs(token) {
				return
			}
		}
		p.nextToken()
	}
}

// ParseProgram parses the entire VCL program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
		Declarations: []ast.Declaration{},
	}

	// Set program reference for subroutine merging
	p.program = program

	// Parse VCL version declaration (required for main files, optional for includes)
	if p.currentTokenIs(lexer.VCL_KW) {
		// Collect any leading comments for the version declaration
		leading := p.consumeComments()
		program.VCLVersion = p.parseVCLVersionDecl()
		if program.VCLVersion == nil {
			return program
		}
		// Attach comments to version declaration
		if program.VCLVersion != nil {
			p.attachCommentsToNode(program.VCLVersion, leading)
		}
		p.nextToken() // Move past the semicolon
	} else if !p.config.AllowMissingVersion {
		p.addError("VCL program must start with version declaration")
		return program
	}

	// Parse declarations
	for !p.currentTokenIs(lexer.EOF) && !p.maxErrorsReached {
		// Collect leading comments before each declaration
		leading := p.consumeComments()

		decl := p.parseDeclaration()
		if decl != nil {
			// Attach comments to the declaration
			p.attachCommentsToNode(decl, leading)
			program.Declarations = append(program.Declarations, decl)
		}
		// Note: if decl is nil, leading comments are discarded (error recovery)

		// Don't advance token if we're at EOF
		if !p.currentTokenIs(lexer.EOF) {
			p.nextToken()
		}
	}

	program.EndPos = p.currentToken.End
	return program
}

// attachCommentsToNode attaches leading and trailing comments to a node
func (p *Parser) attachCommentsToNode(node ast.Node, leading []ast.Comment) {
	// Check if node is nil or if the underlying value is nil
	if node == nil || reflect.ValueOf(node).IsNil() {
		return
	}
	trailing := p.extractTrailingComment()
	if len(leading) > 0 || trailing != nil {
		nodeComments := &ast.NodeComments{
			Leading:  leading,
			Trailing: trailing,
		}
		node.SetComments(nodeComments)
	}
}

// findSubroutineDecl searches for an existing subroutine declaration by name
// in the current program's declarations. Returns the SubDecl if found, nil otherwise.
func (p *Parser) findSubroutineDecl(name string) *ast.SubDecl {
	if p.program == nil {
		return nil
	}
	for _, decl := range p.program.Declarations {
		if subDecl, ok := decl.(*ast.SubDecl); ok {
			if subDecl.Name == name {
				return subDecl
			}
		}
	}
	return nil
}

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() ast.Declaration {
	if p.maxErrorsReached {
		return nil
	}

	if p.panicMode {
		p.skipToSynchronizationPoint(
			lexer.BACKEND_KW, lexer.SUB_KW, lexer.ACL_KW,
			lexer.PROBE_KW, lexer.IMPORT_KW, lexer.INCLUDE_KW,
			lexer.RBRACE, lexer.EOF,
		)
		p.synchronize()
		if p.currentTokenIs(lexer.EOF) || p.currentTokenIs(lexer.RBRACE) {
			return nil
		}
	}

	switch p.currentToken.Type {
	case lexer.IMPORT_KW:
		return p.parseImportDecl()
	case lexer.INCLUDE_KW:
		return p.parseIncludeDecl()
	case lexer.BACKEND_KW:
		return p.parseBackendDecl()
	case lexer.PROBE_KW:
		return p.parseProbeDecl()
	case lexer.ACL_KW:
		return p.parseACLDecl()
	case lexer.SUB_KW:
		subDecl := p.parseSubDecl()
		// Check for nil to avoid adding nil declarations (e.g., merged subroutines)
		if subDecl == nil {
			return nil
		}
		return subDecl
	default:
		p.reportError(fmt.Sprintf("unexpected token %s", p.currentToken.Type))
		return nil
	}
}

// parseVCLVersionDecl parses a VCL version declaration
func (p *Parser) parseVCLVersionDecl() *ast.VCLVersionDecl {
	decl := &ast.VCLVersionDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectToken(lexer.VCL_KW) {
		return nil
	}

	if !p.expectPeek(lexer.FNUM) {
		if !p.currentTokenIs(lexer.CNUM) {
			p.addError("expected version number")
			return nil
		}
	}

	decl.Version = p.currentToken.Value
	decl.EndPos = p.currentToken.End

	if !p.expectPeek(lexer.SEMICOLON) {
		return nil
	}

	return decl
}

// parseImportDecl parses an import declaration
func (p *Parser) parseImportDecl() *ast.ImportDecl {
	decl := &ast.ImportDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	decl.Module = p.currentToken.Value

	// Check for optional alias
	if p.peekTokenIs(lexer.ID) {
		p.nextToken()
		decl.Alias = p.currentToken.Value
	}

	decl.EndPos = p.currentToken.End

	// Consume semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return decl
}

// parseIncludeDecl parses an include declaration
func (p *Parser) parseIncludeDecl() *ast.IncludeDecl {
	decl := &ast.IncludeDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.CSTR) {
		return nil
	}

	// Remove quotes from string literal
	decl.Path = strings.Trim(p.currentToken.Value, `"`)
	decl.EndPos = p.currentToken.End

	// Consume semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return decl
}
