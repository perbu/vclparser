package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/varnish/vclparser/ast"
	"github.com/varnish/vclparser/lexer"
)

// Operator precedence levels
const (
	_ int = iota
	LOWEST
	LOGICAL_OR  // ||
	LOGICAL_AND // &&
	EQUALITY    // ==, !=
	COMPARISON  // <, >, <=, >=
	REGEX       // ~, !~
	TERM        // +, -
	FACTOR      // *, /, %
	UNARY       // !, -, +
	CALL        // function()
	INDEX       // array[index]
	MEMBER      // obj.prop
)

// Precedence map for operators
var precedences = map[lexer.TokenType]int{
	lexer.COR:      LOGICAL_OR,
	lexer.CAND:     LOGICAL_AND,
	lexer.EQ:       EQUALITY,
	lexer.NEQ:      EQUALITY,
	lexer.LT:       COMPARISON,
	lexer.GT:       COMPARISON,
	lexer.LEQ:      COMPARISON,
	lexer.GEQ:      COMPARISON,
	lexer.TILDE:    REGEX,
	lexer.NOMATCH:  REGEX,
	lexer.PLUS:     TERM,
	lexer.MINUS:    TERM,
	lexer.MULTIPLY: FACTOR,
	lexer.DIVIDE:   FACTOR,
	lexer.PERCENT:  FACTOR,
	lexer.LPAREN:   CALL,
	lexer.DOT:      MEMBER,
}

// peekPrecedence returns the precedence of the peek token
func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}
	return LOWEST
}

// currentPrecedence returns the precedence of the current token
func (p *Parser) currentPrecedence() int {
	if p, ok := precedences[p.currentToken.Type]; ok {
		return p
	}
	return LOWEST
}

// parseExpression parses expressions using Pratt parsing
func (p *Parser) parseExpression() ast.Expression {
	return p.parseExpressionWithPrecedence(LOWEST)
}

// parseExpressionWithPrecedence implements a Pratt parser (top-down operator precedence parser)
// using the precedence climbing algorithm. This approach elegantly handles operator precedence
// without deep recursion or backtracking.
//
// The algorithm works by:
// 1. Parse a prefix expression (operand, unary operator, etc.)
// 2. Enter a loop that continues while the next operator has higher precedence than our minimum
// 3. Parse the infix expression (consuming the operator and right operand)
// 4. The right operand parsing respects precedence through recursive calls
//
// Example parsing "a + b * c":
// - Start with precedence=LOWEST, parse "a" as left operand
// - See "+", its precedence (TERM) > LOWEST, so enter loop
// - Parse infix: parseInfixExpression creates BinaryExpression(a, +, right)
// - To parse right side of +, call parseExpressionWithPrecedence(TERM+1)
// - Parse "b", see "*", its precedence (FACTOR) > TERM+1, so enter nested loop
// - Result: BinaryExpression(a, +, BinaryExpression(b, *, c))
//
// The precedence parameter acts as "left-binding power" - operators with higher
// precedence than this value will be consumed by this call, while lower precedence
// operators are left for parent calls to handle.
//
// Termination conditions check for syntactic boundaries where expressions end:
// semicolons (statement end), parentheses/braces (grouping end), commas (argument separator).
func (p *Parser) parseExpressionWithPrecedence(precedence int) ast.Expression {
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	for !p.peekTokenIs(lexer.SEMICOLON) && !p.peekTokenIs(lexer.RPAREN) &&
		!p.peekTokenIs(lexer.RBRACE) && !p.peekTokenIs(lexer.COMMA) &&
		precedence < p.peekPrecedence() {
		if left == nil {
			break
		}
		left = p.parseInfixExpression(left)
		if left == nil {
			return nil
		}
	}

	return left
}

// parsePrefixExpression parses prefix expressions
func (p *Parser) parsePrefixExpression() ast.Expression {
	switch p.currentToken.Type {
	case lexer.ID:
		return p.parseIdentifier()
	// Keywords can also be used as identifiers in some contexts
	case lexer.HASH_KW, lexer.PASS_KW, lexer.PIPE_KW, lexer.FETCH_KW,
		lexer.HIT_KW, lexer.MISS_KW, lexer.DELIVER_KW, lexer.PURGE_KW,
		lexer.SYNTH_KW, lexer.ABANDON_KW, lexer.RETRY_KW, lexer.OK_KW, lexer.FAIL_KW,
		lexer.ERROR_KW, lexer.RESTART_KW, lexer.ACL_KW, lexer.LOOKUP_KW, lexer.VCL_KW:
		return &ast.Identifier{
			BaseNode: ast.BaseNode{
				StartPos: p.currentToken.Start,
				EndPos:   p.currentToken.End,
			},
			Name: p.currentToken.Value,
		}
	case lexer.CNUM:
		// Check if this number is followed by a time unit (like "30s")
		if p.isNumberFollowedByTimeUnit() {
			return p.parseTimeExpressionFromNumber()
		}
		return p.parseIntegerLiteral()
	case lexer.FNUM:
		// Check if this float number is followed by a time unit (like "1.5s")
		if p.isNumberFollowedByTimeUnit() {
			return p.parseTimeExpressionFromNumber()
		}
		return p.parseFloatLiteral()
	case lexer.CSTR:
		return p.parseStringLiteral()
	case lexer.BANG, lexer.MINUS, lexer.PLUS:
		return p.parseUnaryExpression()
	case lexer.LPAREN:
		return p.parseGroupedExpression()
	case lexer.LBRACE:
		return p.parseObjectExpression()
	default:
		// Try to parse as time/duration/IP literal
		if p.isTimeOrDurationLiteral() {
			return p.parseTimeExpression()
		}
		if p.isIPLiteral() {
			return p.parseIPExpression()
		}

		p.addError("unexpected token in expression: " + p.currentToken.Type.String())
		return nil
	}
}

// parseInfixExpression parses infix expressions
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	switch p.peekToken.Type {
	case lexer.COR, lexer.CAND, lexer.EQ, lexer.NEQ, lexer.LT, lexer.GT,
		lexer.LEQ, lexer.GEQ, lexer.PLUS, lexer.MINUS, lexer.MULTIPLY,
		lexer.DIVIDE, lexer.PERCENT:
		return p.parseBinaryExpression(left)
	case lexer.TILDE, lexer.NOMATCH:
		return p.parseRegexMatchExpression(left)
	case lexer.LPAREN:
		return p.parseCallExpression(left)
	case lexer.DOT:
		return p.parseMemberExpression(left)
	default:
		return left
	}
}

// parseIdentifier parses an identifier
func (p *Parser) parseIdentifier() *ast.Identifier {
	return &ast.Identifier{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Name: p.currentToken.Value,
	}
}

// parseIntegerLiteral parses an integer literal
func (p *Parser) parseIntegerLiteral() *ast.IntegerLiteral {
	value, err := strconv.ParseInt(p.currentToken.Value, 0, 64)
	if err != nil {
		p.addError("could not parse " + p.currentToken.Value + " as integer")
		return nil
	}

	return &ast.IntegerLiteral{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Value: value,
	}
}

// parseFloatLiteral parses a float literal
func (p *Parser) parseFloatLiteral() *ast.FloatLiteral {
	value, err := strconv.ParseFloat(p.currentToken.Value, 64)
	if err != nil {
		p.addError("could not parse " + p.currentToken.Value + " as float")
		return nil
	}

	return &ast.FloatLiteral{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Value: value,
	}
}

// parseStringLiteral parses a string literal
func (p *Parser) parseStringLiteral() *ast.StringLiteral {
	// Remove quotes from string literal
	value := strings.Trim(p.currentToken.Value, `"`)

	return &ast.StringLiteral{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Value: value,
	}
}

// parseUnaryExpression parses a unary expression
func (p *Parser) parseUnaryExpression() *ast.UnaryExpression {
	expr := &ast.UnaryExpression{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
		Operator: p.currentToken.Value,
	}

	p.nextToken() // move past operator
	expr.Operand = p.parseExpressionWithPrecedence(UNARY)
	expr.EndPos = p.currentToken.End

	return expr
}

// parseGroupedExpression parses a parenthesized expression
func (p *Parser) parseGroupedExpression() *ast.ParenthesizedExpression {
	expr := &ast.ParenthesizedExpression{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past '('
	expr.Expression = p.parseExpression()

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	expr.EndPos = p.currentToken.End
	return expr
}

// parseBinaryExpression parses a binary expression
func (p *Parser) parseBinaryExpression(left ast.Expression) *ast.BinaryExpression {
	if left == nil {
		p.addError("left expression is nil")
		return nil
	}

	expr := &ast.BinaryExpression{
		BaseNode: ast.BaseNode{
			StartPos: left.Start(),
		},
		Left: left,
	}

	precedence := p.currentPrecedence()
	p.nextToken() // move to operator
	expr.Operator = p.currentToken.Value
	p.nextToken() // move past operator

	expr.Right = p.parseExpressionWithPrecedence(precedence)
	expr.EndPos = p.currentToken.End

	return expr
}

// parseRegexMatchExpression parses regex match expressions
func (p *Parser) parseRegexMatchExpression(left ast.Expression) *ast.RegexMatchExpression {
	expr := &ast.RegexMatchExpression{
		BaseNode: ast.BaseNode{
			StartPos: left.Start(),
		},
		Left: left,
	}

	p.nextToken() // move to operator
	expr.Operator = p.currentToken.Value
	p.nextToken() // move past operator

	expr.Right = p.parseExpressionWithPrecedence(REGEX)
	expr.EndPos = p.currentToken.End

	return expr
}

// parseCallExpression parses a function call expression
func (p *Parser) parseCallExpression(fn ast.Expression) *ast.CallExpression {
	if fn == nil {
		p.addError("function expression is nil")
		return nil
	}

	expr := &ast.CallExpression{
		BaseNode: ast.BaseNode{
			StartPos: fn.Start(),
		},
		Function:       fn,
		NamedArguments: make(map[string]ast.Expression),
	}

	p.nextToken() // move to '('

	// Handle arguments if present
	if !p.peekTokenIs(lexer.RPAREN) {
		p.nextToken() // move past '(' to the first argument's token

		// Phase 1: Parse positional arguments until we see "name =" pattern
		for !p.currentTokenIs(lexer.RPAREN) {
			// Check if this is the start of named arguments (ID followed by =)
			if p.isNamedArgument() {
				break
			}

			// Parse positional argument
			arg := p.parseExpression()
			if arg == nil {
				p.addError("failed to parse function argument")
				return nil
			}
			expr.Arguments = append(expr.Arguments, arg)

			// Break if we hit closing paren or check for comma
			if p.peekTokenIs(lexer.RPAREN) {
				break
			}
			if !p.peekTokenIs(lexer.COMMA) {
				p.addError("expected ',' or ')' after function argument")
				return nil
			}
			p.nextToken() // move to ','
			p.nextToken() // move past ','
		}

		// Phase 2: Parse named arguments
		for p.currentTokenIs(lexer.ID) && p.peekTokenIs(lexer.ASSIGN) {
			// Parse named argument
			argName := p.currentToken.Value
			p.nextToken() // move to '='
			p.nextToken() // move past '='

			// Check for duplicate named argument
			if _, exists := expr.NamedArguments[argName]; exists {
				p.addError(fmt.Sprintf("argument '%s' already used", argName))
				return nil
			}

			arg := p.parseExpression()
			if arg == nil {
				p.addError("failed to parse named argument value")
				return nil
			}
			expr.NamedArguments[argName] = arg

			// Break if we hit closing paren or check for comma
			if p.peekTokenIs(lexer.RPAREN) {
				break
			}
			if !p.peekTokenIs(lexer.COMMA) {
				p.addError("expected ',' or ')' after named argument")
				return nil
			}
			p.nextToken() // move to ','
			p.nextToken() // move past ','
		}
	}

	// After parsing arguments, we MUST find and consume the closing parenthesis
	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	expr.EndPos = p.currentToken.End
	return expr
}

// parseMemberExpression parses member access expressions
func (p *Parser) parseMemberExpression(obj ast.Expression) *ast.MemberExpression {
	expr := &ast.MemberExpression{
		BaseNode: ast.BaseNode{
			StartPos: obj.Start(),
		},
		Object: obj,
	}

	p.nextToken() // move to '.'
	p.nextToken() // move past '.'

	expr.Property = p.parseIdentifier()
	expr.EndPos = p.currentToken.End

	return expr
}

// parseObjectExpression parses object literals (for backend properties)
func (p *Parser) parseObjectExpression() *ast.ObjectExpression {
	expr := &ast.ObjectExpression{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
		},
	}

	p.nextToken() // move past '{'

	for !p.currentTokenIs(lexer.RBRACE) && !p.currentTokenIs(lexer.EOF) {
		if p.currentTokenIs(lexer.COMMENT) {
			p.nextToken()
			continue
		}

		// Skip empty lines or extra whitespace
		if p.currentTokenIs(lexer.SEMICOLON) {
			p.nextToken()
			continue
		}

		prop := &ast.Property{
			BaseNode: ast.BaseNode{
				StartPos: p.currentToken.Start,
			},
		}

		// Parse key - in VCL, object properties start with a dot (e.g., .url)
		if p.currentTokenIs(lexer.DOT) {
			p.nextToken() // move past '.'
			if !p.currentTokenIs(lexer.ID) {
				p.addError("expected property name after '.'")
				return nil
			}
			// Create an identifier for the property name (without the dot)
			prop.Key = &ast.Identifier{
				BaseNode: ast.BaseNode{
					StartPos: p.currentToken.Start,
					EndPos:   p.currentToken.End,
				},
				Name: p.currentToken.Value,
			}
		} else {
			// Fallback to parsing as a general expression
			prop.Key = p.parseExpression()
		}

		if !p.expectPeek(lexer.ASSIGN) {
			return nil
		}

		p.nextToken() // move past '='
		prop.Value = p.parseExpression()
		prop.EndPos = p.currentToken.End

		expr.Properties = append(expr.Properties, prop)

		// VCL uses semicolons to separate object properties
		if p.peekTokenIs(lexer.SEMICOLON) {
			p.nextToken() // move to ';'
			prop.EndPos = p.currentToken.End
			p.nextToken() // move past ';' to next property or '}'
		} else {
			p.nextToken() // move to next token if no semicolon
		}
	}

	if !p.currentTokenIs(lexer.RBRACE) {
		p.addError("expected '}' to close object expression")
		return nil
	}

	expr.EndPos = p.currentToken.End
	return expr
}

// parseTimeExpression parses time/duration expressions
func (p *Parser) parseTimeExpression() *ast.TimeExpression {
	return &ast.TimeExpression{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Value: p.currentToken.Value,
	}
}

// parseIPExpression parses IP address expressions
func (p *Parser) parseIPExpression() *ast.IPExpression {
	return &ast.IPExpression{
		BaseNode: ast.BaseNode{
			StartPos: p.currentToken.Start,
			EndPos:   p.currentToken.End,
		},
		Value: p.currentToken.Value,
	}
}

// Helper functions to detect literal types

// isNumberFollowedByTimeUnit checks if current CNUM/FNUM token is followed by a time unit
func (p *Parser) isNumberFollowedByTimeUnit() bool {
	// Support both integer and float numbers
	if p.currentToken.Type != lexer.CNUM && p.currentToken.Type != lexer.FNUM {
		return false
	}

	// Check if next token is a time unit identifier
	if p.peekToken.Type != lexer.ID {
		return false
	}

	// Use the new duration validation utility
	return IsDurationUnit(p.peekToken.Value)
}

// parseTimeExpressionFromNumber parses time expressions from number + unit (e.g., "30" + "s")
func (p *Parser) parseTimeExpressionFromNumber() *ast.TimeExpression {
	numberValue := p.currentToken.Value
	startPos := p.currentToken.Start

	p.nextToken() // move to time unit
	unitValue := p.currentToken.Value
	endPos := p.currentToken.End

	return &ast.TimeExpression{
		BaseNode: ast.BaseNode{
			StartPos: startPos,
			EndPos:   endPos,
		},
		Value: numberValue + unitValue, // combine "30" + "s" = "30s"
	}
}

// isTimeOrDurationLiteral checks if current token looks like a time/duration literal
func (p *Parser) isTimeOrDurationLiteral() bool {
	value := p.currentToken.Value
	if p.currentToken.Type != lexer.ID {
		return false
	}

	// Use the new duration validation utility to check for complete duration strings
	return ValidateDurationString(value)
}

// isIPLiteral checks if current token looks like an IP address
func (p *Parser) isIPLiteral() bool {
	value := p.currentToken.Value
	if p.currentToken.Type != lexer.ID {
		return false
	}

	// Simple check for IPv4 pattern (more sophisticated validation could be added)
	parts := strings.Split(value, ".")
	if len(parts) == 4 {
		for _, part := range parts {
			if _, err := strconv.Atoi(part); err != nil {
				return false
			}
		}
		return true
	}

	// Simple check for IPv6 (contains colons)
	return strings.Contains(value, ":")
}

// isNamedArgument checks if current token is the start of a named argument (ID followed by =)
func (p *Parser) isNamedArgument() bool {
	return p.currentTokenIs(lexer.ID) && p.peekTokenIs(lexer.ASSIGN)
}
