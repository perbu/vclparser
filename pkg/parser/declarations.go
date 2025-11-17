package parser

import (
	"github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
	"github.com/perbu/vclparser/pkg/types"
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
			// Error recovery: skip to next property or closing brace
			p.skipToSynchronizationPoint(lexer.DOT, lexer.RBRACE, lexer.SEMICOLON)
			if p.currentTokenIs(lexer.SEMICOLON) {
				p.nextToken() // consume semicolon and continue
			}
		}
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	decl.EndPos = p.currentToken.End
	return decl
}

// parseBackendProperty parses individual backend configuration properties.
// Handles special cases like probe properties that can contain nested object
// expressions, while treating other properties as simple value assignments.
// All properties must start with dot notation (.url, .port, .probe).
func (p *Parser) parseBackendProperty() *ast.BackendProperty {
	if !p.currentTokenIs(lexer.DOT) {
		p.reportError("backend property must start with '.'")
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
		p.reportError("expected property name after '.'")
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
	// Use parsePropertyValue to support implicit string concatenation
	// (multiple string literals in a row, especially for .request property)
	prop.Value = p.parsePropertyValue()

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

// parseSubDecl parses subroutine declarations and registers them in the symbol table.
// VCL allows multiple definitions of the same subroutine, which are merged together
// in the order they appear. This is commonly used with built-in subroutines
// (vcl_recv, vcl_backend_fetch) across multiple include files.
func (p *Parser) parseSubDecl() *ast.SubDecl {
	startPos := p.currentToken.Start

	if !p.expectPeek(lexer.ID) {
		return nil
	}

	subroutineName := p.currentToken.Value

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	// Parse the subroutine body
	body := p.parseBlockStatement()
	endPos := p.currentToken.End

	// Check if this subroutine already exists
	existing := p.findSubroutineDecl(subroutineName)

	// Register in symbol table (allows duplicates for subroutines)
	symbol := &types.Symbol{
		Name:     subroutineName,
		Kind:     types.SymbolSubroutine,
		Type:     types.Void,
		Position: startPos,
	}
	_ = p.symbolTable.Define(symbol) // Won't error for duplicate subroutines

	if existing != nil {
		// Merge: append new body statements to existing subroutine
		if body != nil && existing.Body != nil && len(body.Statements) > 0 {
			existing.Body.Statements = append(existing.Body.Statements, body.Statements...)
			// Update end position to reflect merged content
			existing.EndPos = endPos
		}
		// Return nil so this declaration isn't added to program.Declarations again
		return nil
	}

	// First definition - create new declaration
	decl := &ast.SubDecl{
		BaseNode: ast.BaseNode{
			StartPos: startPos,
			EndPos:   endPos,
		},
		Name: subroutineName,
		Body: body,
	}

	return decl
}
