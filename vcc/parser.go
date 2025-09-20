package vcc

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// VCCLexer interface to abstract lexer implementation
type VCCLexer interface {
	NextToken() Token
}

// Parser parses VCC files into Module definitions
type Parser struct {
	lexer        VCCLexer
	errors       []string
	currentToken Token
}

// NewParser creates a new VCC parser
func NewParser(r io.Reader) *Parser {
	lexer := NewSimpleLexer(r)
	p := &Parser{
		lexer:  lexer,
		errors: []string{},
	}
	p.nextToken() // Initialize current token
	return p
}

// nextToken advances to the next token
func (p *Parser) nextToken() {
	p.currentToken = p.lexer.NextToken()
}

// Parse parses the VCC file and returns a Module
func (p *Parser) Parse() (*Module, error) {
	module := &Module{
		Functions: []Function{},
		Objects:   []Object{},
		Events:    []Event{},
	}

	for p.currentToken.Type != EOF {
		switch p.currentToken.Type {
		case MODULE:
			if err := p.parseModuleDecl(module); err != nil {
				p.addError(err.Error())
			}
		case FUNCTION:
			if function, err := p.parseFunction(); err != nil {
				p.addError(err.Error())
			} else {
				module.Functions = append(module.Functions, *function)
			}
		case OBJECT:
			if object, err := p.parseObject(); err != nil {
				p.addError(err.Error())
			} else {
				module.Objects = append(module.Objects, *object)
			}
		case EVENT:
			if event, err := p.parseEvent(); err != nil {
				p.addError(err.Error())
			} else {
				module.Events = append(module.Events, *event)
			}
		case ABI:
			if err := p.parseABI(module); err != nil {
				p.addError(err.Error())
			}
		case COMMENT:
			// Skip comments
			p.nextToken()
		case DESCRIPTION:
			// Parse module description
			if desc, err := p.parseDescription(); err != nil {
				p.addError(err.Error())
			} else {
				module.Description = desc
			}
		default:
			// Skip unknown tokens
			p.nextToken()
		}
	}

	if len(p.errors) > 0 {
		return module, fmt.Errorf("parse errors: %s", strings.Join(p.errors, "; "))
	}

	return module, nil
}

// parseModuleDecl parses $Module directive
func (p *Parser) parseModuleDecl(module *Module) error {
	// $Module name version "description"
	p.nextToken() // consume $Module

	// Parse module name
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected module name, got %s", p.currentToken.Type)
	}
	module.Name = p.currentToken.Literal
	p.nextToken()

	// Parse version
	if p.currentToken.Type != NUMBER {
		return fmt.Errorf("expected version number, got %s", p.currentToken.Type)
	}
	version, err := strconv.Atoi(p.currentToken.Literal)
	if err != nil {
		return fmt.Errorf("invalid version number: %s", p.currentToken.Literal)
	}
	module.Version = version
	p.nextToken()

	// Parse description (optional)
	if p.currentToken.Type == STRING {
		module.Description = p.currentToken.Literal
		p.nextToken()
	}

	return nil
}

// parseFunction parses $Function directive
func (p *Parser) parseFunction() (*Function, error) {
	// $Function RETURN_TYPE name(PARAM_TYPE param, ...)
	p.nextToken() // consume $Function

	function := &Function{
		Parameters:   []Parameter{},
		Examples:     []string{},
		Restrictions: []string{},
	}

	// Parse function signature: RETURN_TYPE name(params)
	if err := p.parseFunctionSignatureTokens(function); err != nil {
		return nil, err
	}

	// Parse description and examples
	for p.currentToken.Type != EOF {
		// Stop if we hit another directive
		if p.currentToken.Type == MODULE || p.currentToken.Type == FUNCTION ||
			p.currentToken.Type == OBJECT || p.currentToken.Type == METHOD ||
			p.currentToken.Type == EVENT || p.currentToken.Type == ABI {
			break
		}

		if p.currentToken.Type == RESTRICT {
			p.nextToken()
			restriction := p.readUntilNewline()
			function.Restrictions = append(function.Restrictions, restriction)
		} else {
			// Read description text
			line := p.readUntilNewline()
			if strings.TrimSpace(line) != "" {
				if function.Description == "" {
					function.Description = line
				} else {
					function.Description += "\n" + line
				}
			}
		}
	}

	return function, nil
}

// parseObject parses $Object directive
func (p *Parser) parseObject() (*Object, error) {
	// $Object name(PARAM_TYPE param, ...)
	p.nextToken() // consume $Object

	object := &Object{
		Constructor: []Parameter{},
		Methods:     []Method{},
		Examples:    []string{},
	}

	// Parse object signature: name(params)
	if err := p.parseObjectSignatureTokens(object); err != nil {
		return nil, err
	}

	// Parse description and methods
	for p.currentToken.Type != EOF {
		token := p.currentToken

		// Stop if we hit another top-level directive
		if token.Type == MODULE || token.Type == FUNCTION || token.Type == OBJECT ||
			token.Type == EVENT || token.Type == ABI {
			break
		}

		if token.Type == METHOD {
			method, err := p.parseMethod()
			if err != nil {
				return nil, err
			}
			object.Methods = append(object.Methods, *method)
		} else {
			// Read description text
			line := p.readUntilNewline()
			if strings.TrimSpace(line) != "" {
				if object.Description == "" {
					object.Description = line
				} else {
					object.Description += "\n" + line
				}
			}
		}
	}

	return object, nil
}

// parseMethod parses $Method directive
func (p *Parser) parseMethod() (*Method, error) {
	// $Method RETURN_TYPE .name(PARAM_TYPE param, ...)
	p.nextToken() // consume $Method

	method := &Method{
		Parameters:   []Parameter{},
		Examples:     []string{},
		Restrictions: []string{},
	}

	// Parse method signature: RETURN_TYPE .name(params)
	if err := p.parseMethodSignatureTokens(method); err != nil {
		return nil, err
	}

	// Parse description and restrictions
	for p.currentToken.Type != EOF {
		token := p.currentToken

		// Stop if we hit another directive
		if token.Type == MODULE || token.Type == FUNCTION || token.Type == OBJECT ||
			token.Type == METHOD || token.Type == EVENT || token.Type == ABI {
			break
		}

		if token.Type == RESTRICT {
			p.nextToken()
			restriction := p.readUntilNewline()
			method.Restrictions = append(method.Restrictions, restriction)
		} else {
			// Read description text
			line := p.readUntilNewline()
			if strings.TrimSpace(line) != "" {
				if method.Description == "" {
					method.Description = line
				} else {
					method.Description += "\n" + line
				}
			}
		}
	}

	return method, nil
}

// parseEvent parses $Event directive
func (p *Parser) parseEvent() (*Event, error) {
	// $Event name
	p.nextToken() // consume $Event

	if p.currentToken.Type != IDENT {
		return nil, fmt.Errorf("expected event name, got %s", p.currentToken.Type)
	}

	event := &Event{
		Name: p.currentToken.Literal,
	}
	p.nextToken()

	return event, nil
}

// parseABI parses $ABI directive
func (p *Parser) parseABI(module *Module) error {
	// $ABI strict
	p.nextToken() // consume $ABI

	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected ABI specification, got %s", p.currentToken.Type)
	}

	module.ABI = p.currentToken.Literal
	p.nextToken()

	return nil
}

// parseDescription parses a DESCRIPTION section
func (p *Parser) parseDescription() (string, error) {
	p.nextToken() // consume DESCRIPTION

	var description strings.Builder

	// Read until we hit another directive or end of file
	for p.currentToken.Type != EOF {
		token := p.currentToken

		// Stop if we hit a directive
		if token.Type == MODULE || token.Type == FUNCTION || token.Type == OBJECT ||
			token.Type == METHOD || token.Type == EVENT || token.Type == ABI {
			break
		}

		line := p.readUntilNewline()
		if strings.TrimSpace(line) != "" {
			description.WriteString(line)
			description.WriteString("\n")
		}
	}

	return description.String(), nil
}

// parseFunctionSignatureTokens parses function signature from tokens
func (p *Parser) parseFunctionSignatureTokens(function *Function) error {
	// Parse return type
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected return type, got %s", p.currentToken.Type)
	}
	returnType, _, err := ParseVCCType(p.currentToken.Literal)
	if err != nil {
		return fmt.Errorf("invalid return type: %v", err)
	}
	function.ReturnType = returnType
	p.nextToken()

	// Parse function name
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected function name, got %s", p.currentToken.Type)
	}
	function.Name = p.currentToken.Literal
	p.nextToken()

	// Parse parameters
	if p.currentToken.Type == LPAREN {
		p.nextToken() // consume (

		for p.currentToken.Type != RPAREN && p.currentToken.Type != EOF {
			param, err := p.parseParameterTokens()
			if err != nil {
				return fmt.Errorf("invalid parameter: %v", err)
			}
			function.Parameters = append(function.Parameters, param)

			if p.currentToken.Type == COMMA {
				p.nextToken() // consume comma
			}
		}

		if p.currentToken.Type == RPAREN {
			p.nextToken() // consume )
		}
	}

	return nil
}

// parseParameterTokens parses a parameter from tokens
func (p *Parser) parseParameterTokens() (Parameter, error) {
	var param Parameter

	// Check for optional parameter syntax with square brackets
	hasOpenBracket := false
	if p.currentToken.Type == LBRACKET {
		param.Optional = true
		hasOpenBracket = true
		p.nextToken() // consume [
	}

	// Parse type (might be ENUM{...})
	if p.currentToken.Type == IDENT && p.currentToken.Literal == "ENUM" {
		// Handle ENUM type
		p.nextToken()
		if p.currentToken.Type == LBRACE {
			enumType := "ENUM{"
			p.nextToken()
			for p.currentToken.Type != RBRACE && p.currentToken.Type != EOF {
				enumType += p.currentToken.Literal
				p.nextToken()
				if p.currentToken.Type == COMMA {
					enumType += ","
					p.nextToken()
				}
			}
			if p.currentToken.Type == RBRACE {
				enumType += "}"
				p.nextToken()
			}

			vccType, enum, err := ParseVCCType(enumType)
			if err != nil {
				return param, err
			}
			param.Type = vccType
			param.Enum = enum
		}
	} else if p.currentToken.Type == IDENT {
		// Regular type
		vccType, enum, err := ParseVCCType(p.currentToken.Literal)
		if err != nil {
			return param, err
		}
		param.Type = vccType
		param.Enum = enum
		p.nextToken()
	} else {
		return param, fmt.Errorf("expected parameter type at line %d:%d, got %s",
			p.currentToken.Line, p.currentToken.Column, p.currentToken.Type)
	}

	// Parse parameter name (optional in some VCC files)
	if p.currentToken.Type == IDENT && p.currentToken.Literal != "," {
		param.Name = p.currentToken.Literal
		p.nextToken()
	}

	// For optional parameters, check if we hit the closing bracket immediately after the name
	if hasOpenBracket && p.currentToken.Type == RBRACKET {
		p.nextToken() // consume ]
		return param, nil
	}

	// Check for default value
	if p.currentToken.Type == EQUALS {
		p.nextToken()
		if p.currentToken.Type == STRING || p.currentToken.Type == IDENT || p.currentToken.Type == NUMBER {
			param.DefaultValue = p.currentToken.Literal
			param.Optional = true
			p.nextToken()
		}
	}

	// Consume closing bracket if we had an opening bracket
	if hasOpenBracket {
		if p.currentToken.Type == RBRACKET {
			p.nextToken() // consume ]
		} else {
			return param, fmt.Errorf("expected closing bracket ']' for optional parameter at line %d:%d, got %s",
				p.currentToken.Line, p.currentToken.Column, p.currentToken.Type)
		}
	}

	return param, nil
}

// parseFunctionSignature parses a function signature string
func (p *Parser) parseFunctionSignature(function *Function, signature string) error {
	// Parse: RETURN_TYPE name(PARAM_TYPE param, ...)
	re := regexp.MustCompile(`^\s*(\w+(?:\s*\{[^}]*\})?)\s+(\w+)\s*\(([^)]*)\)\s*$`)
	matches := re.FindStringSubmatch(signature)

	if len(matches) != 4 {
		return fmt.Errorf("invalid function signature: %s", signature)
	}

	// Parse return type
	returnType, _, err := ParseVCCType(matches[1])
	if err != nil {
		return fmt.Errorf("invalid return type in function signature: %v", err)
	}
	function.ReturnType = returnType

	// Parse function name
	function.Name = matches[2]

	// Parse parameters
	if params := strings.TrimSpace(matches[3]); params != "" {
		function.Parameters, err = p.parseParameters(params)
		if err != nil {
			return fmt.Errorf("invalid parameters in function signature: %v", err)
		}
	}

	return nil
}

// parseObjectSignatureTokens parses object signature from tokens
func (p *Parser) parseObjectSignatureTokens(object *Object) error {
	// Parse object name
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected object name, got %s", p.currentToken.Type)
	}
	object.Name = p.currentToken.Literal
	p.nextToken()

	// Parse constructor parameters
	if p.currentToken.Type == LPAREN {
		p.nextToken() // consume (

		for p.currentToken.Type != RPAREN && p.currentToken.Type != EOF {
			param, err := p.parseParameterTokens()
			if err != nil {
				return fmt.Errorf("invalid constructor parameter: %v", err)
			}
			object.Constructor = append(object.Constructor, param)

			if p.currentToken.Type == COMMA {
				p.nextToken() // consume comma
			}
		}

		if p.currentToken.Type == RPAREN {
			p.nextToken() // consume )
		}
	}

	return nil
}

// parseObjectSignature parses an object signature string
func (p *Parser) parseObjectSignature(object *Object, signature string) error {
	// Parse: name(PARAM_TYPE param, ...)
	re := regexp.MustCompile(`^\s*(\w+)\s*\(([^)]*)\)\s*$`)
	matches := re.FindStringSubmatch(signature)

	if len(matches) != 3 {
		return fmt.Errorf("invalid object signature: %s", signature)
	}

	// Parse object name
	object.Name = matches[1]

	// Parse constructor parameters
	if params := strings.TrimSpace(matches[2]); params != "" {
		var err error
		object.Constructor, err = p.parseParameters(params)
		if err != nil {
			return fmt.Errorf("invalid constructor parameters: %v", err)
		}
	}

	return nil
}

// parseMethodSignatureTokens parses method signature from tokens
func (p *Parser) parseMethodSignatureTokens(method *Method) error {
	// Parse return type
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected return type, got %s", p.currentToken.Type)
	}
	returnType, _, err := ParseVCCType(p.currentToken.Literal)
	if err != nil {
		return fmt.Errorf("invalid return type: %v", err)
	}
	method.ReturnType = returnType
	p.nextToken()

	// Parse method name (should start with .)
	if p.currentToken.Type == DOT {
		p.nextToken()
	}
	if p.currentToken.Type != IDENT {
		return fmt.Errorf("expected method name, got %s", p.currentToken.Type)
	}
	method.Name = p.currentToken.Literal
	p.nextToken()

	// Parse parameters
	if p.currentToken.Type == LPAREN {
		p.nextToken() // consume (

		for p.currentToken.Type != RPAREN && p.currentToken.Type != EOF {
			param, err := p.parseParameterTokens()
			if err != nil {
				return fmt.Errorf("invalid parameter: %v", err)
			}
			method.Parameters = append(method.Parameters, param)

			if p.currentToken.Type == COMMA {
				p.nextToken() // consume comma
			}
		}

		if p.currentToken.Type == RPAREN {
			p.nextToken() // consume )
		}
	}

	return nil
}

// parseMethodSignature parses a method signature string
func (p *Parser) parseMethodSignature(method *Method, signature string) error {
	// Parse: RETURN_TYPE .name(PARAM_TYPE param, ...)
	re := regexp.MustCompile(`^\s*(\w+(?:\s*\{[^}]*\})?)\s+\.(\w+)\s*\(([^)]*)\)\s*$`)
	matches := re.FindStringSubmatch(signature)

	if len(matches) != 4 {
		return fmt.Errorf("invalid method signature: %s", signature)
	}

	// Parse return type
	returnType, _, err := ParseVCCType(matches[1])
	if err != nil {
		return fmt.Errorf("invalid return type in method signature: %v", err)
	}
	method.ReturnType = returnType

	// Parse method name
	method.Name = matches[2]

	// Parse parameters
	if params := strings.TrimSpace(matches[3]); params != "" {
		method.Parameters, err = p.parseParameters(params)
		if err != nil {
			return fmt.Errorf("invalid parameters in method signature: %v", err)
		}
	}

	return nil
}

// parseParameters parses parameter list string
func (p *Parser) parseParameters(paramStr string) ([]Parameter, error) {
	var parameters []Parameter

	// Split by commas, but handle ENUMs with braces
	parts := p.splitParameters(paramStr)

	for _, part := range parts {
		param, err := p.parseParameter(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		parameters = append(parameters, param)
	}

	return parameters, nil
}

// parseParameter parses a single parameter definition
func (p *Parser) parseParameter(paramStr string) (Parameter, error) {
	// Handle: [TYPE name=default] or TYPE name=default or TYPE name or ENUM{...} name=default

	var param Parameter

	// Check for optional parameter syntax with square brackets
	paramStr = strings.TrimSpace(paramStr)
	if strings.HasPrefix(paramStr, "[") && strings.HasSuffix(paramStr, "]") {
		param.Optional = true
		paramStr = strings.TrimSpace(paramStr[1 : len(paramStr)-1])
	}

	// Check for default value
	parts := strings.SplitN(paramStr, "=", 2)
	if len(parts) == 2 {
		param.DefaultValue = strings.TrimSpace(strings.Trim(parts[1], `"`))
		param.Optional = true
		paramStr = strings.TrimSpace(parts[0])
	}

	// Split type and name
	tokens := strings.Fields(paramStr)
	if len(tokens) < 2 {
		return param, fmt.Errorf("invalid parameter format: %s", paramStr)
	}

	// Handle ENUM types that might have spaces
	if strings.HasPrefix(tokens[0], "ENUM") {
		// Find the closing brace
		enumEnd := 0
		for i, token := range tokens {
			if strings.Contains(token, "}") {
				enumEnd = i
				break
			}
		}

		if enumEnd > 0 {
			enumType := strings.Join(tokens[:enumEnd+1], " ")
			paramType, enum, err := ParseVCCType(enumType)
			if err != nil {
				return param, err
			}
			param.Type = paramType
			param.Enum = enum
			param.Name = tokens[enumEnd+1]
		} else {
			return param, fmt.Errorf("malformed ENUM type: %s", paramStr)
		}
	} else {
		// Regular type
		paramType, enum, err := ParseVCCType(tokens[0])
		if err != nil {
			return param, err
		}
		param.Type = paramType
		param.Enum = enum
		param.Name = tokens[1]
	}

	return param, nil
}

// splitParameters splits parameter string respecting braces and brackets
func (p *Parser) splitParameters(paramStr string) []string {
	var parts []string
	var current strings.Builder
	braceCount := 0
	bracketCount := 0

	for _, char := range paramStr {
		switch char {
		case '{':
			braceCount++
			current.WriteRune(char)
		case '}':
			braceCount--
			current.WriteRune(char)
		case '[':
			bracketCount++
			current.WriteRune(char)
		case ']':
			bracketCount--
			current.WriteRune(char)
		case ',':
			if braceCount == 0 && bracketCount == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// readUntilNewline reads the rest of the current line
func (p *Parser) readUntilNewline() string {
	var line strings.Builder

	for p.currentToken.Type != EOF {
		token := p.currentToken

		// Check if we've hit a new directive (which starts a new logical line)
		if token.Type == MODULE || token.Type == FUNCTION || token.Type == OBJECT ||
			token.Type == METHOD || token.Type == EVENT || token.Type == ABI || token.Type == RESTRICT {
			break
		}

		if token.Type != COMMENT {
			if line.Len() > 0 {
				line.WriteString(" ")
			}
			line.WriteString(token.Literal)
		}

		p.nextToken()

		// If we hit EOF, break
		if p.currentToken.Type == EOF {
			break
		}
	}

	return line.String()
}

// addError adds an error to the error list
func (p *Parser) addError(msg string) {
	p.errors = append(p.errors, msg)
}

// Errors returns the list of parse errors
func (p *Parser) Errors() []string {
	return p.errors
}
