package parser

import (
	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
)

// parseBackendDecl parses a backend declaration
func (p *Parser) parseBackendDecl() *ast.BackendDecl {
	decl := &ast.BackendDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	decl.Name = p.currentToken.Value

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse backend properties
	p.nextToken() // move past '{'

	for !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		prop := p.parseBackendProperty()
		if prop != nil {
			decl.Properties = append(decl.Properties, prop)
			// parseBackendProperty already advances past the semicolon
		} else {
			// Skip to next token if parsing failed
			p.nextToken()
		}
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	decl.EndPos = p.currentToken.End
	return decl
}

// parseBackendProperty parses a backend property
func (p *Parser) parseBackendProperty() *ast.BackendProperty {
	if !p.currentTokenIs(lexer.DOT) {
		p.addError("backend property must start with '.'")
		return nil
	}

	prop := &ast.BackendProperty{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move to the property name token

	// Property name can be either an ID or certain keywords like "probe"
	if p.currentTokenIs(lexer.ID) || p.currentTokenIs(lexer.PROBE_KW) {
		prop.Name = p.currentToken.Value
	} else {
		p.addError("expected property name after '.'")
		return nil
	}

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken() // move to value

	// If the property is a probe and the value is a block, parse it as an object expression
	if prop.Name == "probe" && p.currentTokenIs(lexer.LBRACE) {
		prop.Value = p.parseObjectExpression()
	} else {
		// Otherwise, parse it as a normal expression (e.g., a string or identifier)
		prop.Value = p.parseExpression()
	}

	// Move past the value to the semicolon
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to semicolon
		prop.EndPos = p.currentToken.End
		p.nextToken() // move past semicolon
	} else {
		prop.EndPos = p.currentToken.End
	}

	return prop
}

// parseProbeDecl parses a probe declaration
func (p *Parser) parseProbeDecl() *ast.ProbeDecl {
	decl := &ast.ProbeDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	decl.Name = p.currentToken.Value

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse probe properties
	p.nextToken() // move past '{'

	for !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		prop := p.parseProbeProperty()
		if prop != nil {
			decl.Properties = append(decl.Properties, prop)
			// parseProbeProperty already advances past the semicolon
		} else {
			// Skip to next token if parsing failed
			p.nextToken()
		}
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	decl.EndPos = p.currentToken.End
	return decl
}

// parseProbeProperty parses a probe property
func (p *Parser) parseProbeProperty() *ast.ProbeProperty {
	if !p.currentTokenIs(lexer.DOT) {
		p.addError("probe property must start with '.'")
		return nil
	}

	prop := &ast.ProbeProperty{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	prop.Name = p.currentToken.Value

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken() // move to value
	prop.Value = p.parseExpression()

	// Move past the value to the semicolon
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to semicolon
		prop.EndPos = p.currentToken.End
		p.nextToken() // move past semicolon
	} else {
		prop.EndPos = p.currentToken.End
	}

	return prop
}

// parseACLDecl parses an ACL declaration
func (p *Parser) parseACLDecl() *ast.ACLDecl {
	decl := &ast.ACLDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // Move past 'acl'

	// ACL name can be an identifier or certain keywords
	if !p.currentTokenIs(lexer.ID) && !p.currentToken.Type.IsKeyword() {
		p.addError("expected ACL name")
		return nil
	}

	decl.Name = p.currentToken.Value

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse ACL entries
	p.nextToken() // move past '{'

	for !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		entry := p.parseACLEntry()
		if entry != nil {
			decl.Entries = append(decl.Entries, entry)
		}

		p.nextToken()
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	decl.EndPos = p.currentToken.End
	return decl
}

// parseACLEntry parses an ACL entry
func (p *Parser) parseACLEntry() *ast.ACLEntry {
	entry := &ast.ACLEntry{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	// Check for negation
	if p.currentTokenIs(lexer.BANG) {
		entry.Negated = true
		p.nextToken()
	}

	// Parse the network specification
	entry.Network = p.parseExpression()
	entry.EndPos = p.currentToken.End

	// Consume semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return entry
}

// parseSubDecl parses a subroutine declaration
func (p *Parser) parseSubDecl() *ast.SubDecl {
	decl := &ast.SubDecl{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	decl.Name = p.currentToken.Value

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse the subroutine body
	decl.Body = p.parseBlockStatement()
	decl.EndPos = p.currentToken.End

	return decl
}
