package parser

import (
	"fmt"
	"strings"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

// Parser implements a recursive descent parser for VCL
type Parser struct {
	lexer    *lexer.Lexer
	errors   []DetailedError
	input    string // Store original VCL source for error context
	filename string // Store filename for error reporting

	currentToken lexer.Token
	peekToken    lexer.Token
}

// New creates a new parser
func New(l *lexer.Lexer, input, filename string) *Parser {
	p := &Parser{
		lexer:    l,
		errors:   []DetailedError{},
		input:    input,
		filename: filename,
	}

	// Read two tokens, so currentToken and peekToken are both set
	p.nextToken()
	p.nextToken()

	return p
}

// Parse parses the input and returns the AST
func Parse(input, filename string) (*ast.Program, error) {
	l := lexer.New(input, filename)
	p := New(l, input, filename)
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

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.currentToken = p.peekToken
	p.peekToken = p.lexer.NextToken()

	// Skip comments during parsing
	for p.peekToken.Type == lexer.COMMENT {
		p.peekToken = p.lexer.NextToken()
	}
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
	p.addError(fmt.Sprintf("expected next token to be %s, got %s", t, p.peekToken.Type))
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

// ParseProgram parses the entire VCL program
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
		Declarations: []ast.Declaration{},
	}

	// Skip any initial comments
	for p.currentTokenIs(lexer.COMMENT) {
		p.nextToken()
	}

	// Parse VCL version declaration (required first)
	if p.currentTokenIs(lexer.VCL_KW) {
		program.VCLVersion = p.parseVCLVersionDecl()
		if program.VCLVersion == nil {
			return program
		}
		p.nextToken() // Move past the semicolon
	} else {
		p.addError("VCL program must start with version declaration")
		return program
	}

	// Parse declarations
	for !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		decl := p.parseDeclaration()
		if decl != nil {
			program.Declarations = append(program.Declarations, decl)
		}

		// Don't advance token if we're at EOF
		if !p.currentTokenIs(lexer.EOF) {
			p.nextToken()
		}
	}

	program.EndPos = p.currentToken.End
	return program
}

// parseDeclaration parses a top-level declaration
func (p *Parser) parseDeclaration() ast.Declaration {
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
		return p.parseSubDecl()
	default:
		p.addError(fmt.Sprintf("unexpected token %s", p.currentToken.Type))
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
