package vcc

import (
	"bufio"
	"io"
	"strings"
	"unicode"
)

// SimpleLexer is a simpler, more robust VCC lexer
type SimpleLexer struct {
	input       *bufio.Scanner
	currentLine string
	line        int
	position    int
	column      int
}

// NewSimpleLexer creates a new simple VCC lexer
func NewSimpleLexer(r io.Reader) *SimpleLexer {
	return &SimpleLexer{
		input: bufio.NewScanner(r),
		line:  0,
	}
}

// NextToken reads the next token
func (l *SimpleLexer) NextToken() Token {
	for {
		// Skip whitespace and empty lines
		l.skipWhitespace()

		// If we're at end of line, try to read next line
		if l.position >= len(l.currentLine) {
			if !l.nextLine() {
				return Token{Type: EOF, Line: l.line, Column: l.column}
			}
			continue
		}

		ch := l.currentChar()
		startColumn := l.column

		// Handle specific characters
		switch ch {
		case '$':
			return l.readDirective()
		case '#':
			return l.readComment()
		case '"':
			return l.readString()
		case '\'':
			return l.readString()
		case '(':
			l.advance()
			return Token{Type: LPAREN, Literal: "(", Line: l.line, Column: startColumn}
		case ')':
			l.advance()
			return Token{Type: RPAREN, Literal: ")", Line: l.line, Column: startColumn}
		case '{':
			l.advance()
			return Token{Type: LBRACE, Literal: "{", Line: l.line, Column: startColumn}
		case '}':
			l.advance()
			return Token{Type: RBRACE, Literal: "}", Line: l.line, Column: startColumn}
		case '[':
			l.advance()
			return Token{Type: LBRACKET, Literal: "[", Line: l.line, Column: startColumn}
		case ']':
			l.advance()
			return Token{Type: RBRACKET, Literal: "]", Line: l.line, Column: startColumn}
		case ',':
			l.advance()
			return Token{Type: COMMA, Literal: ",", Line: l.line, Column: startColumn}
		case '=':
			l.advance()
			return Token{Type: EQUALS, Literal: "=", Line: l.line, Column: startColumn}
		case '.':
			l.advance()
			return Token{Type: DOT, Literal: ".", Line: l.line, Column: startColumn}
		case ';':
			l.advance()
			return Token{Type: SEMICOLON, Literal: ";", Line: l.line, Column: startColumn}
		default:
			if unicode.IsLetter(rune(ch)) || ch == '_' {
				return l.readIdentifier()
			} else if unicode.IsDigit(rune(ch)) || ch == '-' {
				return l.readNumber()
			} else {
				l.advance()
				return Token{Type: ILLEGAL, Literal: string(ch), Line: l.line, Column: startColumn}
			}
		}
	}
}

// currentChar returns the current character
func (l *SimpleLexer) currentChar() byte {
	if l.position >= len(l.currentLine) {
		return 0
	}
	return l.currentLine[l.position]
}

// advance moves to the next character
func (l *SimpleLexer) advance() {
	if l.position < len(l.currentLine) {
		l.position++
		l.column++
	}
}

// nextLine moves to the next line
func (l *SimpleLexer) nextLine() bool {
	if l.input.Scan() {
		l.currentLine = l.input.Text()
		l.line++
		l.position = 0
		l.column = 0
		return true
	}
	return false
}

// skipWhitespace skips whitespace characters
func (l *SimpleLexer) skipWhitespace() {
	for l.position < len(l.currentLine) && unicode.IsSpace(rune(l.currentChar())) {
		l.advance()
	}
}

// readDirective reads a VCC directive ($Module, $Function, etc.)
func (l *SimpleLexer) readDirective() Token {
	startColumn := l.column
	startPos := l.position
	l.advance() // skip $

	// Read the directive name
	for l.position < len(l.currentLine) && (unicode.IsLetter(rune(l.currentChar())) || l.currentChar() == '_') {
		l.advance()
	}

	literal := l.currentLine[startPos:l.position]
	tokenType := l.lookupDirective(literal)

	return Token{Type: tokenType, Literal: literal, Line: l.line, Column: startColumn}
}

// readComment reads a comment line
func (l *SimpleLexer) readComment() Token {
	startColumn := l.column
	literal := l.currentLine[l.position:]
	l.position = len(l.currentLine) // consume rest of line

	return Token{Type: COMMENT, Literal: literal, Line: l.line, Column: startColumn}
}

// readString reads a quoted string literal
func (l *SimpleLexer) readString() Token {
	startColumn := l.column
	quoteChar := l.currentChar() // remember which quote type we're using
	l.advance()                  // skip opening quote

	var value strings.Builder
	for l.position < len(l.currentLine) && l.currentChar() != quoteChar {
		if l.currentChar() == '\\' {
			l.advance()
			if l.position < len(l.currentLine) {
				value.WriteByte(l.currentChar())
				l.advance()
			}
		} else {
			value.WriteByte(l.currentChar())
			l.advance()
		}
	}

	if l.position < len(l.currentLine) && l.currentChar() == quoteChar {
		l.advance() // skip closing quote
	}

	return Token{Type: STRING, Literal: value.String(), Line: l.line, Column: startColumn}
}

// readIdentifier reads an identifier or keyword
func (l *SimpleLexer) readIdentifier() Token {
	startColumn := l.column
	startPos := l.position

	for l.position < len(l.currentLine) && (unicode.IsLetter(rune(l.currentChar())) || unicode.IsDigit(rune(l.currentChar())) || l.currentChar() == '_') {
		l.advance()
	}

	literal := l.currentLine[startPos:l.position]
	tokenType := l.lookupIdent(literal)

	return Token{Type: tokenType, Literal: literal, Line: l.line, Column: startColumn}
}

// readNumber reads a numeric literal
func (l *SimpleLexer) readNumber() Token {
	startColumn := l.column
	startPos := l.position

	// Handle optional minus sign
	if l.currentChar() == '-' {
		l.advance()
	}

	for l.position < len(l.currentLine) && (unicode.IsDigit(rune(l.currentChar())) || l.currentChar() == '.') {
		l.advance()
	}

	// Check for duration suffix (s, m, h, d, w, y, ms)
	if l.position < len(l.currentLine) {
		ch := l.currentChar()
		if ch == 's' || ch == 'm' || ch == 'h' || ch == 'd' || ch == 'w' || ch == 'y' {
			l.advance()
			// Check for "ms" (milliseconds)
			if ch == 'm' && l.position < len(l.currentLine) && l.currentChar() == 's' {
				l.advance()
			}
		}
	}

	literal := l.currentLine[startPos:l.position]

	return Token{Type: NUMBER, Literal: literal, Line: l.line, Column: startColumn}
}

// lookupDirective maps directive strings to token types
func (l *SimpleLexer) lookupDirective(literal string) TokenType {
	switch literal {
	case "$Module":
		return MODULE
	case "$Function":
		return FUNCTION
	case "$Object":
		return OBJECT
	case "$Method":
		return METHOD
	case "$Event":
		return EVENT
	case "$Restrict":
		return RESTRICT
	case "$ABI":
		return ABI
	case "$License":
		return LICENSE
	default:
		return IDENT
	}
}

// lookupIdent maps identifier strings to token types
func (l *SimpleLexer) lookupIdent(literal string) TokenType {
	switch literal {
	case "DESCRIPTION":
		return DESCRIPTION
	case "Example":
		return EXAMPLE
	case "DEFAULT":
		return DEFAULT
	case "true", "false":
		return BOOL_LIT
	default:
		return IDENT
	}
}

// All returns all tokens (useful for testing)
func (l *SimpleLexer) All() []Token {
	var tokens []Token

	for {
		token := l.NextToken()
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	return tokens
}
