package parser

import (
	"fmt"

	ast2 "github.com/perbu/vclparser/pkg/ast"
	"github.com/perbu/vclparser/pkg/lexer"
)

// parseStatement parses a statement
func (p *Parser) parseStatement() ast2.Statement {
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
func (p *Parser) parseBlockStatement() *ast2.BlockStatement {
	stmt := &ast2.BlockStatement{
		BaseNode: ast2.BaseNode{
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
		stmt.Statements = append(stmt.Statements, statement)

		p.nextToken()
	}

	if !p.expectToken(lexer.RBRACE) {
		return nil
	}

	stmt.EndPos = p.currentToken.End
	return stmt
}

// parseIfStatement parses an if statement
func (p *Parser) parseIfStatement() *ast2.IfStatement {
	stmt := &ast2.IfStatement{
		BaseNode: ast2.BaseNode{
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
func (p *Parser) parseSetStatement() *ast2.SetStatement {
	stmt := &ast2.SetStatement{
		BaseNode: ast2.BaseNode{
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

	// Set end position safely
	if p.currentToken.Type != lexer.EOF {
		stmt.EndPos = p.currentToken.End
	} else {
		stmt.EndPos = stmt.Value.End()
	}

	// Consume the semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to semicolon
		stmt.EndPos = p.currentToken.End
	}

	return stmt
}

// parseUnsetStatement parses an unset statement
func (p *Parser) parseUnsetStatement() *ast2.UnsetStatement {
	stmt := &ast2.UnsetStatement{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'unset'
	stmt.Variable = p.parseExpression()

	// Set end position safely
	if p.currentToken.Type != lexer.EOF {
		stmt.EndPos = p.currentToken.End
	} else {
		stmt.EndPos = stmt.Variable.End()
	}

	// Consume the semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to semicolon
		stmt.EndPos = p.currentToken.End
	}

	return stmt
}

// parseCallStatement parses a call statement
func (p *Parser) parseCallStatement() *ast2.CallStatement {
	stmt := &ast2.CallStatement{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past 'call'

	// Expect an identifier (subroutine name)
	if !p.currentTokenIs(lexer.ID) {
		p.addError(fmt.Sprintf("expected identifier after 'call', got %s", p.currentToken.Type))
		return nil
	}

	// Create identifier for the subroutine name
	stmt.Function = &ast2.Identifier{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Name: p.currentToken.Value,
	}

	stmt.EndPos = p.currentToken.End

	// Advance past the identifier before checking for semicolon
	p.nextToken()
	p.skipSemicolon()
	return stmt
}

// parseReturnStatement parses a return statement
func (p *Parser) parseReturnStatement() *ast2.ReturnStatement {
	stmt := &ast2.ReturnStatement{
		BaseNode: ast2.BaseNode{
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
	}

	stmt.EndPos = p.currentToken.End

	// Consume semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// parseSyntheticStatement parses a synthetic statement
func (p *Parser) parseSyntheticStatement() *ast2.SyntheticStatement {
	stmt := &ast2.SyntheticStatement{
		BaseNode: ast2.BaseNode{
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
func (p *Parser) parseErrorStatement() *ast2.ErrorStatement {
	stmt := &ast2.ErrorStatement{
		BaseNode: ast2.BaseNode{
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
func (p *Parser) parseRestartStatement() *ast2.RestartStatement {
	stmt := &ast2.RestartStatement{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
	}

	p.skipSemicolon()
	return stmt
}

// parseCSourceStatement parses a C source statement
func (p *Parser) parseCSourceStatement() *ast2.CSourceStatement {
	stmt := &ast2.CSourceStatement{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Code: p.currentToken.Value,
	}

	return stmt
}

// parseNewStatement parses a new statement for VMOD object instantiation
func (p *Parser) parseNewStatement() *ast2.NewStatement {
	stmt := &ast2.NewStatement{
		BaseNode: ast2.BaseNode{
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

	// Debug: check token position before parsing constructor
	// fmt.Printf("DEBUG: About to parse constructor, current: %s, peek: %s\n", p.currentToken.Type, p.peekToken.Type)

	stmt.Constructor = p.parseExpression()

	if stmt.Constructor == nil {
		p.addError("expected constructor expression after '=' in new statement")
		return nil
	}

	// Set end position safely
	if p.currentToken.Type != lexer.EOF {
		stmt.EndPos = p.currentToken.End
	} else {
		stmt.EndPos = stmt.Constructor.End()
	}

	// Consume the semicolon if present
	if p.peekTokenIs(lexer.SEMICOLON) {
		p.nextToken() // move to semicolon
		stmt.EndPos = p.currentToken.End
	}

	return stmt
}

// parseExpressionStatement parses an expression statement
func (p *Parser) parseExpressionStatement() *ast2.ExpressionStatement {
	stmt := &ast2.ExpressionStatement{
		BaseNode: ast2.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	stmt.Expression = p.parseExpression()
	if stmt.Expression == nil {
		return nil
	}

	// Use current token end position as a safer alternative to calling End()
	// This avoids the panic that occurs when CallExpression.End() is called
	if _, ok := stmt.Expression.(*ast2.CallExpression); ok {
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
