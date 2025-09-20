package lexer

// Lexer tokenizes VCL source code
type Lexer struct {
	input    string
	filename string
	pos      int  // current position in input (points to current char)
	readPos  int  // current reading position in input (after current char)
	ch       byte // current char under examination
	line     int  // current line number (1-indexed)
	column   int  // current column number (1-indexed)
}

// New creates a new lexer instance
func New(input, filename string) *Lexer {
	l := &Lexer{
		input:    input,
		filename: filename,
		line:     1,
		column:   1,
	}
	l.readChar() // Initialize first character
	return l
}

// readChar reads the next character and advances position in input
func (l *Lexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0 // EOF
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++

	if l.ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
}

// peekChar returns the next character without advancing position
func (l *Lexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

// currentPosition returns the current position
func (l *Lexer) currentPosition() Position {
	return Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.pos,
	}
}

// NextToken scans the input and returns the next token
func (l *Lexer) NextToken() Token {
	var tok Token

	l.skipWhitespace()

	tok.Start = l.currentPosition()
	tok.Filename = l.filename

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(EQ)
		} else {
			tok = l.makeToken(ASSIGN)
		}
	case '+':
		switch l.peekChar() {
		case '+':
			tok = l.makeTwoCharToken(INC)
		case '=':
			tok = l.makeTwoCharToken(INCR)
		default:
			tok = l.makeToken(PLUS)
		}
	case '-':
		switch l.peekChar() {
		case '-':
			tok = l.makeTwoCharToken(DEC)
		case '=':
			tok = l.makeTwoCharToken(DECR)
		default:
			tok = l.makeToken(MINUS)
		}
	case '*':
		if l.peekChar() == '=' {
			tok = l.makeTwoCharToken(MUL)
		} else {
			tok = l.makeToken(MULTIPLY)
		}
	case '/':
		switch l.peekChar() {
		case '=':
			tok = l.makeTwoCharToken(DIV)
		case '/':
			// Single line comment
			tok = l.readLineComment()
		case '*':
			// Multi-line comment
			tok = l.readBlockComment()
		default:
			tok = l.makeToken(DIVIDE)
		}
	case '!':
		switch l.peekChar() {
		case '=':
			tok = l.makeTwoCharToken(NEQ)
		case '~':
			tok = l.makeTwoCharToken(NOMATCH)
		default:
			tok = l.makeToken(BANG)
		}
	case '<':
		switch l.peekChar() {
		case '=':
			tok = l.makeTwoCharToken(LEQ)
		case '<':
			tok = l.makeTwoCharToken(SHL)
		default:
			tok = l.makeToken(LT)
		}
	case '>':
		switch l.peekChar() {
		case '=':
			tok = l.makeTwoCharToken(GEQ)
		case '>':
			tok = l.makeTwoCharToken(SHR)
		default:
			tok = l.makeToken(GT)
		}
	case '&':
		if l.peekChar() == '&' {
			tok = l.makeTwoCharToken(CAND)
		} else {
			tok = l.makeToken(AMPERSAND)
		}
	case '|':
		if l.peekChar() == '|' {
			tok = l.makeTwoCharToken(COR)
		} else {
			tok = l.makeToken(PIPE)
		}
	case '{':
		tok = l.makeToken(LBRACE)
	case '}':
		tok = l.makeToken(RBRACE)
	case '(':
		tok = l.makeToken(LPAREN)
	case ')':
		tok = l.makeToken(RPAREN)
	case ';':
		tok = l.makeToken(SEMICOLON)
	case ',':
		tok = l.makeToken(COMMA)
	case '.':
		tok = l.makeToken(DOT)
	case '%':
		tok = l.makeToken(PERCENT)
	case '~':
		tok = l.makeToken(TILDE)
	case '"':
		tok = l.readString()
	case 'C':
		// Check for C{ ... }C block
		if l.peekChar() == '{' {
			tok = l.readCBlock()
		} else {
			tok = l.readIdentifier()
		}
	case '#':
		// Shell-style comment
		tok = l.readLineComment()
	case 0:
		tok.Type = EOF
		tok.Value = ""
	default:
		if isLetter(l.ch) {
			tok = l.readIdentifier()
			return tok // Early return to avoid advancing twice
		} else if isDigit(l.ch) {
			tok = l.readNumber()
			return tok // Early return to avoid advancing twice
		} else {
			tok = l.makeToken(ILLEGAL)
		}
	}

	tok.End = l.currentPosition()
	l.readChar()
	return tok
}

// makeToken creates a token with the current character
func (l *Lexer) makeToken(tokenType TokenType) Token {
	return Token{
		Type:     tokenType,
		Value:    string(l.ch),
		Start:    l.currentPosition(),
		Filename: l.filename,
	}
}

// makeTwoCharToken creates a token with current and next character
func (l *Lexer) makeTwoCharToken(tokenType TokenType) Token {
	ch := l.ch
	l.readChar()
	return Token{
		Type:     tokenType,
		Value:    string(ch) + string(l.ch),
		Start:    l.currentPosition(),
		Filename: l.filename,
	}
}

// readIdentifier reads an identifier or keyword
func (l *Lexer) readIdentifier() Token {
	start := l.currentPosition()
	startPos := l.pos

	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}

	value := l.input[startPos:l.pos]
	tokenType := LookupKeyword(value)

	return Token{
		Type:     tokenType,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// readNumber reads a number (integer or float)
func (l *Lexer) readNumber() Token {
	start := l.currentPosition()
	startPos := l.pos
	tokenType := CNUM

	// Read integer part
	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		tokenType = FNUM
		l.readChar() // consume '.'

		// Read fractional part
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	// Check for scientific notation
	if l.ch == 'e' || l.ch == 'E' {
		tokenType = FNUM
		l.readChar()

		if l.ch == '+' || l.ch == '-' {
			l.readChar()
		}

		for isDigit(l.ch) {
			l.readChar()
		}
	}

	value := l.input[startPos:l.pos]

	return Token{
		Type:     tokenType,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// readString reads a string literal
func (l *Lexer) readString() Token {
	start := l.currentPosition()
	startPos := l.pos

	l.readChar() // consume opening quote

	for l.ch != '"' && l.ch != 0 {
		if l.ch == '\\' {
			l.readChar() // consume backslash
			if l.ch != 0 {
				l.readChar() // consume escaped character
			}
		} else {
			l.readChar()
		}
	}

	if l.ch == 0 {
		return Token{
			Type:     ILLEGAL,
			Value:    "unterminated string",
			Start:    start,
			End:      l.currentPosition(),
			Filename: l.filename,
		}
	}

	value := l.input[startPos : l.pos+1] // Include closing quote

	return Token{
		Type:     CSTR,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// readCBlock reads a C code block (C{ ... }C)
func (l *Lexer) readCBlock() Token {
	start := l.currentPosition()
	startPos := l.pos

	l.readChar() // consume 'C'
	l.readChar() // consume '{'

	depth := 1
	for depth > 0 && l.ch != 0 {
		if l.ch == '}' && l.peekChar() == 'C' {
			l.readChar() // consume '}'
			l.readChar() // consume 'C'
			depth--
		} else if l.ch == 'C' && l.peekChar() == '{' {
			l.readChar() // consume 'C'
			l.readChar() // consume '{'
			depth++
		} else {
			l.readChar()
		}
	}

	if depth > 0 {
		return Token{
			Type:     ILLEGAL,
			Value:    "unterminated C block",
			Start:    start,
			End:      l.currentPosition(),
			Filename: l.filename,
		}
	}

	value := l.input[startPos:l.pos]

	return Token{
		Type:     CSRC,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// readLineComment reads a single-line comment
func (l *Lexer) readLineComment() Token {
	start := l.currentPosition()
	startPos := l.pos

	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}

	value := l.input[startPos:l.pos]

	return Token{
		Type:     COMMENT,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// readBlockComment reads a multi-line comment
func (l *Lexer) readBlockComment() Token {
	start := l.currentPosition()
	startPos := l.pos

	l.readChar() // consume first '/'
	l.readChar() // consume '*'

	for {
		if l.ch == 0 {
			return Token{
				Type:     ILLEGAL,
				Value:    "unterminated comment",
				Start:    start,
				End:      l.currentPosition(),
				Filename: l.filename,
			}
		}

		if l.ch == '*' && l.peekChar() == '/' {
			l.readChar() // consume '*'
			l.readChar() // consume '/'
			break
		}

		l.readChar()
	}

	value := l.input[startPos:l.pos]

	return Token{
		Type:     COMMENT,
		Value:    value,
		Start:    start,
		End:      l.currentPosition(),
		Filename: l.filename,
	}
}

// skipWhitespace skips whitespace characters
func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

// Helper functions
func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

// TokenizeAll tokenizes the entire input and returns all tokens
func (l *Lexer) TokenizeAll() []Token {
	var tokens []Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == EOF {
			break
		}
	}

	return tokens
}

// TokenizeAllSkipComments tokenizes the input and returns all tokens except comments
func (l *Lexer) TokenizeAllSkipComments() []Token {
	var tokens []Token

	for {
		tok := l.NextToken()
		if tok.Type != COMMENT {
			tokens = append(tokens, tok)
		}
		if tok.Type == EOF {
			break
		}
	}

	return tokens
}
