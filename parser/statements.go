package parser

import (
	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

// parseStatement parses a statement
func (p *Parser) parseStatement() ast.Statement {
	switch p.currentToken.Type {
	case lexer.IF_KW:
		return p.parseIfStatement()
	case lexer.SET_KW:
		return p.parseSetStatement()
	case lexer.UNSET_KW:
		return p.parseUnsetStatement()
	case lexer.CALL_KW:
		return p.parseCallStatement()
	case lexer.RETURN_KW:
		return p.parseReturnStatement()
	case lexer.SYNTHETIC_KW:
		return p.parseSyntheticStatement()
	case lexer.ERROR_KW:
		return p.parseErrorStatement()
	case lexer.RESTART_KW:
		return p.parseRestartStatement()
	case lexer.NEW_KW:
		return p.parseNewStatement()
	case lexer.LBRACE:
		return p.parseBlockStatement()
	case lexer.CSRC:
		return p.parseCSourceStatement()
	default:
		// Try to parse as expression statement
		return p.parseExpressionStatement()
	}
}

// parseBlockStatement parses a block statement
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	stmt := &ast.BlockStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectToken(lexer.LBRACE) {
		return nil
	}

	p.nextToken() // move past '{'

	for !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		statement := p.parseStatement()
		if statement != nil {
			stmt.Statements = append(stmt.Statements, statement)
		} else {
			// Skip to next semicolon or brace to recover from error
			for !p.currentTokenIs(lexer.SEMICOLON) && !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
				p.nextToken()
			}
		}

		p.nextToken()
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	stmt.EndPos = p.currentToken.End
	return stmt
}

// parseIfStatement parses an if statement
func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	p.nextToken() // move past '('
	stmt.Condition = p.parseExpression()

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	if !p.expectPeek(lexer.LBRACE) {
		return nil
	}

	stmt.Then = p.parseBlockStatement()

	// Check for else clause
	if p.peekTokenIs(lexer.ELSE_KW) || p.peekTokenIs(lexer.ELSEIF_KW) ||
		p.peekTokenIs(lexer.ELSIF_KW) || p.peekTokenIs(lexer.ELIF_KW) {
		p.nextToken() // move to else/elseif token

		if p.currentTokenIs(lexer.ELSE_KW) {
			if p.peekTokenIs(lexer.IF_KW) {
				// else if
				p.nextToken() // move to if
				stmt.Else = p.parseIfStatement()
			} else {
				// else block
				if !p.expectPeek(lexer.LBRACE) {
					return nil
				}
				stmt.Else = p.parseBlockStatement()
			}
		} else {
			// elseif/elsif/elif
			stmt.Else = p.parseIfStatement()
		}
	}

	stmt.EndPos = p.currentToken.End
	return stmt
}

// parseSetStatement parses a set statement
func (p *Parser) parseSetStatement() *ast.SetStatement {
	stmt := &ast.SetStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'set'
	stmt.Variable = p.parseExpression()

	// Parse assignment operator
	if p.peekTokenIs(lexer.ASSIGN) || p.peekTokenIs(lexer.INCR) ||
		p.peekTokenIs(lexer.DECR) || p.peekTokenIs(lexer.MUL) ||
		p.peekTokenIs(lexer.DIV) {
		p.nextToken()
		stmt.Operator = p.currentToken.Value
		p.nextToken()
		stmt.Value = p.parseExpression()
	} else {
		p.addError("expected assignment operator")
		return nil
	}

	// Advance to semicolon if we're not already there
	if !p.currentTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}
	stmt.EndPos = p.currentToken.End
	p.skipSemicolon()
	return stmt
}

// parseUnsetStatement parses an unset statement
func (p *Parser) parseUnsetStatement() *ast.UnsetStatement {
	stmt := &ast.UnsetStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'unset'
	stmt.Variable = p.parseExpression()
	stmt.EndPos = p.currentToken.End

	p.skipSemicolon()
	return stmt
}

// parseCallStatement parses a call statement
func (p *Parser) parseCallStatement() *ast.CallStatement {
	stmt := &ast.CallStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'call'
	stmt.Function = p.parseExpression()
	stmt.EndPos = p.currentToken.End

	p.skipSemicolon()
	return stmt
}

// parseReturnStatement parses a return statement
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // move past 'return'
		p.nextToken() // move past '('
		stmt.Action = p.parseExpression()

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	} else if !p.peekTokenIs(lexer.SEMICOLON) {
		// return with action but no parentheses
		p.nextToken() // move past 'return'
		stmt.Action = p.parseExpression()
	}

	stmt.EndPos = p.currentToken.End

	// Consume semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseSyntheticStatement parses a synthetic statement
func (p *Parser) parseSyntheticStatement() *ast.SyntheticStatement {
	stmt := &ast.SyntheticStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // move past 'synthetic'
		p.nextToken() // move past '('
		stmt.Response = p.parseExpression()

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	} else {
		p.nextToken() // move past 'synthetic'
		stmt.Response = p.parseExpression()
	}

	stmt.EndPos = p.currentToken.End
	p.skipSemicolon()
	return stmt
}

// parseErrorStatement parses an error statement
func (p *Parser) parseErrorStatement() *ast.ErrorStatement {
	stmt := &ast.ErrorStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken() // move past 'error'
		p.nextToken() // move past '('

		// Parse optional code and response
		stmt.Code = p.parseExpression()

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken() // move past ','
			p.nextToken() // move to response
			stmt.Response = p.parseExpression()
		}

		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	}

	stmt.EndPos = p.currentToken.End
	p.skipSemicolon()
	return stmt
}

// parseRestartStatement parses a restart statement
func (p *Parser) parseRestartStatement() *ast.RestartStatement {
	stmt := &ast.RestartStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
	}

	p.skipSemicolon()
	return stmt
}

// parseCSourceStatement parses a C source statement
func (p *Parser) parseCSourceStatement() *ast.CSourceStatement {
	stmt := &ast.CSourceStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Code: p.currentToken.Value,
	}

	return stmt
}

// parseNewStatement parses a new statement for VMOD object instantiation
func (p *Parser) parseNewStatement() *ast.NewStatement {
	stmt := &ast.NewStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'new'
	stmt.Name = p.parseExpression()

	// Parse assignment operator
	if !p.expectPeek(lexer.ASSIGN) {
		p.addError("expected '=' after variable name in new statement")
		return nil
	}

	p.nextToken() // move past '='
	stmt.Constructor = p.parseExpression()

	if stmt.Constructor == nil {
		p.addError("expected constructor expression after '=' in new statement")
		return nil
	}

	stmt.EndPos = p.currentToken.End
	p.skipSemicolon()
	return stmt
}

// parseExpressionStatement parses an expression statement
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	stmt.Expression = p.parseExpression()
	if stmt.Expression == nil {
		return nil
	}

	// Use current token end position as a safer alternative to calling End()
	// This avoids the panic that occurs when CallExpression.End() is called
	if _, ok := stmt.Expression.(*ast.CallExpression); ok {
		stmt.EndPos = p.currentToken.End
	} else {
		stmt.EndPos = stmt.Expression.End()
	}

	// Advance to semicolon
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // Move to semicolon
	}

	return stmt
}
