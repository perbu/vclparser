package vcc

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// TokenType represents the type of VCC token
type TokenType int

const (
	// Special tokens
	EOF TokenType = iota
	ILLEGAL
	COMMENT

	// VCC directives
	MODULE   // $Module
	FUNCTION // $Function
	OBJECT   // $Object
	METHOD   // $Method
	EVENT    // $Event
	RESTRICT // $Restrict
	ABI      // $ABI
	LICENSE  // $License

	// Literals
	IDENT    // identifiers, type names
	STRING   // string literals
	NUMBER   // numeric literals
	BOOL_LIT // true/false

	// Delimiters
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	COMMA     // ,
	EQUALS    // =
	DOT       // .
	SEMICOLON // ;

	// Keywords
	DESCRIPTION // DESCRIPTION
	EXAMPLE     // Example
	DEFAULT     // DEFAULT
)

// Token represents a lexical token
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

// String returns a string representation of the token
func (t Token) String() string {
	return fmt.Sprintf("{%s %q %d:%d}", t.Type.String(), t.Literal, t.Line, t.Column)
}

// String returns a string representation of the token type
func (tt TokenType) String() string {
	switch tt {
	case EOF:
		return "EOF"
	case ILLEGAL:
		return "ILLEGAL"
	case COMMENT:
		return "COMMENT"
	case MODULE:
		return "MODULE"
	case FUNCTION:
		return "FUNCTION"
	case OBJECT:
		return "OBJECT"
	case METHOD:
		return "METHOD"
	case EVENT:
		return "EVENT"
	case RESTRICT:
		return "RESTRICT"
	case ABI:
		return "ABI"
	case LICENSE:
		return "LICENSE"
	case IDENT:
		return "IDENT"
	case STRING:
		return "STRING"
	case NUMBER:
		return "NUMBER"
	case BOOL_LIT:
		return "BOOL_LIT"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LBRACE:
		return "LBRACE"
	case RBRACE:
		return "RBRACE"
	case COMMA:
		return "COMMA"
	case EQUALS:
		return "EQUALS"
	case DOT:
		return "DOT"
	case SEMICOLON:
		return "SEMICOLON"
	case DESCRIPTION:
		return "DESCRIPTION"
	case EXAMPLE:
		return "EXAMPLE"
	case DEFAULT:
		return "DEFAULT"
	default:
		return "UNKNOWN"
	}
}

// Lexer tokenizes VCC files
type Lexer struct {
	input        *bufio.Scanner
	currentToken Token
	peekToken    Token
	line         int
	column       int
	currentLine  string
	position     int // position in current line
}

// NewLexer creates a new VCC lexer
func NewLexer(r io.Reader) *Lexer {
	l := &Lexer{
		input:  bufio.NewScanner(r),
		line:   1,
		column: 0,
	}

	// Read first line
	if l.input.Scan() {
		l.currentLine = l.input.Text()
	}

	// Read two tokens to initialize current and peek
	l.readToken()
	l.readToken()

	return l
}

// CurrentToken returns the current token
func (l *Lexer) CurrentToken() Token {
	return l.currentToken
}

// PeekToken returns the next token without advancing
func (l *Lexer) PeekToken() Token {
	return l.peekToken
}

// NextToken advances to the next token
func (l *Lexer) NextToken() Token {
	l.currentToken = l.peekToken
	l.readToken()
	return l.currentToken
}

// readToken reads the next token from input
func (l *Lexer) readToken() {
	l.skipWhitespace()

	if l.position >= len(l.currentLine) {
		if !l.nextLine() {
			l.peekToken = Token{Type: EOF, Line: l.line, Column: l.column}
			return
		}
		l.skipWhitespace()
	}

	// Handle end of input
	if l.position >= len(l.currentLine) && !l.input.Scan() {
		l.peekToken = Token{Type: EOF, Line: l.line, Column: l.column}
		return
	}

	ch := l.currentChar()
	startColumn := l.column

	switch ch {
	case '(':
		l.peekToken = Token{Type: LPAREN, Literal: "(", Line: l.line, Column: startColumn}
		l.advance()
	case ')':
		l.peekToken = Token{Type: RPAREN, Literal: ")", Line: l.line, Column: startColumn}
		l.advance()
	case '{':
		l.peekToken = Token{Type: LBRACE, Literal: "{", Line: l.line, Column: startColumn}
		l.advance()
	case '}':
		l.peekToken = Token{Type: RBRACE, Literal: "}", Line: l.line, Column: startColumn}
		l.advance()
	case ',':
		l.peekToken = Token{Type: COMMA, Literal: ",", Line: l.line, Column: startColumn}
		l.advance()
	case '=':
		l.peekToken = Token{Type: EQUALS, Literal: "=", Line: l.line, Column: startColumn}
		l.advance()
	case '.':
		l.peekToken = Token{Type: DOT, Literal: ".", Line: l.line, Column: startColumn}
		l.advance()
	case ';':
		l.peekToken = Token{Type: SEMICOLON, Literal: ";", Line: l.line, Column: startColumn}
		l.advance()
	case '"':
		l.peekToken = l.readString()
	case '#':
		l.peekToken = l.readComment()
	case '$':
		l.peekToken = l.readDirective()
	default:
		if unicode.IsLetter(rune(ch)) || ch == '_' {
			l.peekToken = l.readIdentifier()
		} else if unicode.IsDigit(rune(ch)) {
			l.peekToken = l.readNumber()
		} else {
			l.peekToken = Token{Type: ILLEGAL, Literal: string(ch), Line: l.line, Column: startColumn}
			l.advance()
		}
	}
}

// currentChar returns the current character
func (l *Lexer) currentChar() byte {
	if l.position >= len(l.currentLine) {
		return 0
	}
	return l.currentLine[l.position]
}

// advance moves to the next character
func (l *Lexer) advance() {
	l.position++
	l.column++
}

// nextLine moves to the next line
func (l *Lexer) nextLine() bool {
	if l.input.Scan() {
		l.currentLine = l.input.Text()
		l.line++
		l.column = 0
		l.position = 0
		return true
	}
	return false
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.position < len(l.currentLine) && unicode.IsSpace(rune(l.currentChar())) {
		l.advance()
	}
}

// readString reads a quoted string literal
func (l *Lexer) readString() Token {
	startColumn := l.column
	l.advance() // skip opening quote

	var value strings.Builder
	for l.position < len(l.currentLine) && l.currentChar() != '"' {
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

	if l.position < len(l.currentLine) && l.currentChar() == '"' {
		l.advance() // skip closing quote
	}

	return Token{Type: STRING, Literal: value.String(), Line: l.line, Column: startColumn}
}

// readComment reads a comment line
func (l *Lexer) readComment() Token {
	startColumn := l.column
	value := l.currentLine[l.position:]
	l.position = len(l.currentLine) // consume rest of line

	return Token{Type: COMMENT, Literal: value, Line: l.line, Column: startColumn}
}

// readDirective reads a VCC directive ($Module, $Function, etc.)
func (l *Lexer) readDirective() Token {
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

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() Token {
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
func (l *Lexer) readNumber() Token {
	startColumn := l.column
	startPos := l.position

	for l.position < len(l.currentLine) && (unicode.IsDigit(rune(l.currentChar())) || l.currentChar() == '.') {
		l.advance()
	}

	literal := l.currentLine[startPos:l.position]

	return Token{Type: NUMBER, Literal: literal, Line: l.line, Column: startColumn}
}

// lookupDirective maps directive strings to token types
func (l *Lexer) lookupDirective(literal string) TokenType {
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
		return ILLEGAL
	}
}

// lookupIdent maps identifier strings to token types
func (l *Lexer) lookupIdent(literal string) TokenType {
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
func (l *Lexer) All() []Token {
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

// IsAtEOF returns true if the lexer is at end of file
func (l *Lexer) IsAtEOF() bool {
	return l.currentToken.Type == EOF
}
