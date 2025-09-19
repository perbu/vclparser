package lexer

import "fmt"

// TokenType represents the type of a VCL token
type TokenType int

// Position represents a position in the source code
type Position struct {
	Line   int // Line number (1-indexed)
	Column int // Column number (1-indexed)
	Offset int // Byte offset (0-indexed)
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Column)
}

// Token represents a single VCL token
type Token struct {
	Type     TokenType
	Value    string
	Start    Position
	End      Position
	Filename string
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q)", t.Type, t.Value)
}

// Token types based on lib/libvcc/generate.py
const (
	// Special tokens
	ILLEGAL TokenType = iota
	EOF
	COMMENT

	// Literals
	ID   // identifiers and keywords
	CNUM // integer number
	FNUM // floating-point number
	CSTR // string literal
	CSRC // C source code block

	// Multi-character operators (from tokens map in generate.py)
	INC     // ++
	DEC     // --
	CAND    // &&
	COR     // ||
	LEQ     // <=
	EQ      // ==
	NEQ     // !=
	GEQ     // >=
	SHR     // >>
	SHL     // <<
	INCR    // +=
	DECR    // -=
	MUL     // *=
	DIV     // /=
	NOMATCH // !~

	// Single character tokens
	LBRACE    // {
	RBRACE    // }
	LPAREN    // (
	RPAREN    // )
	MULTIPLY  // *
	PLUS      // +
	MINUS     // -
	DIVIDE    // /
	PERCENT   // %
	GT        // >
	LT        // <
	ASSIGN    // =
	SEMICOLON // ;
	BANG      // !
	AMPERSAND // &
	DOT       // .
	PIPE      // |
	TILDE     // ~
	COMMA     // ,

	// Keywords - these will be resolved from ID tokens
	VCL_KW
	BACKEND_KW
	SUB_KW
	PROBE_KW
	ACL_KW
	IMPORT_KW
	INCLUDE_KW
	IF_KW
	ELSE_KW
	ELSEIF_KW
	ELSIF_KW
	ELIF_KW
	SET_KW
	UNSET_KW
	CALL_KW
	RETURN_KW
	SYNTHETIC_KW
	ERROR_KW
	RESTART_KW
	PASS_KW
	PIPE_KW
	HASH_KW
	LOOKUP_KW
	MISS_KW
	HIT_KW
	FETCH_KW
	DELIVER_KW
	PURGE_KW
	SYNTH_KW
	ABANDON_KW
	RETRY_KW
	OK_KW
	FAIL_KW
	NEW_KW
)

// String returns the string representation of a token type
func (t TokenType) String() string {
	switch t {
	case ILLEGAL:
		return "ILLEGAL"
	case EOF:
		return "EOF"
	case COMMENT:
		return "COMMENT"
	case ID:
		return "ID"
	case CNUM:
		return "CNUM"
	case FNUM:
		return "FNUM"
	case CSTR:
		return "CSTR"
	case CSRC:
		return "CSRC"
	case INC:
		return "++"
	case DEC:
		return "--"
	case CAND:
		return "&&"
	case COR:
		return "||"
	case LEQ:
		return "<="
	case EQ:
		return "=="
	case NEQ:
		return "!="
	case GEQ:
		return ">="
	case SHR:
		return ">>"
	case SHL:
		return "<<"
	case INCR:
		return "+="
	case DECR:
		return "-="
	case MUL:
		return "*="
	case DIV:
		return "/="
	case NOMATCH:
		return "!~"
	case LBRACE:
		return "{"
	case RBRACE:
		return "}"
	case LPAREN:
		return "("
	case RPAREN:
		return ")"
	case MULTIPLY:
		return "*"
	case PLUS:
		return "+"
	case MINUS:
		return "-"
	case DIVIDE:
		return "/"
	case PERCENT:
		return "%"
	case GT:
		return ">"
	case LT:
		return "<"
	case ASSIGN:
		return "="
	case SEMICOLON:
		return ";"
	case BANG:
		return "!"
	case AMPERSAND:
		return "&"
	case DOT:
		return "."
	case PIPE:
		return "|"
	case TILDE:
		return "~"
	case COMMA:
		return ","
	case VCL_KW:
		return "vcl"
	case BACKEND_KW:
		return "backend"
	case SUB_KW:
		return "sub"
	case PROBE_KW:
		return "probe"
	case ACL_KW:
		return "acl"
	case IMPORT_KW:
		return "import"
	case INCLUDE_KW:
		return "include"
	case IF_KW:
		return "if"
	case ELSE_KW:
		return "else"
	case ELSEIF_KW:
		return "elseif"
	case ELSIF_KW:
		return "elsif"
	case ELIF_KW:
		return "elif"
	case SET_KW:
		return "set"
	case UNSET_KW:
		return "unset"
	case CALL_KW:
		return "call"
	case RETURN_KW:
		return "return"
	case SYNTHETIC_KW:
		return "synthetic"
	case ERROR_KW:
		return "error"
	case RESTART_KW:
		return "restart"
	case PASS_KW:
		return "pass"
	case PIPE_KW:
		return "pipe"
	case HASH_KW:
		return "hash"
	case LOOKUP_KW:
		return "lookup"
	case MISS_KW:
		return "miss"
	case HIT_KW:
		return "hit"
	case FETCH_KW:
		return "fetch"
	case DELIVER_KW:
		return "deliver"
	case PURGE_KW:
		return "purge"
	case SYNTH_KW:
		return "synth"
	case ABANDON_KW:
		return "abandon"
	case RETRY_KW:
		return "retry"
	case OK_KW:
		return "ok"
	case FAIL_KW:
		return "fail"
	case NEW_KW:
		return "new"
	default:
		return fmt.Sprintf("TokenType(%d)", int(t))
	}
}

// Keywords maps string literals to their token types
var Keywords = map[string]TokenType{
	"vcl":       VCL_KW,
	"backend":   BACKEND_KW,
	"sub":       SUB_KW,
	"probe":     PROBE_KW,
	"acl":       ACL_KW,
	"import":    IMPORT_KW,
	"include":   INCLUDE_KW,
	"if":        IF_KW,
	"else":      ELSE_KW,
	"elseif":    ELSEIF_KW,
	"elsif":     ELSIF_KW,
	"elif":      ELIF_KW,
	"set":       SET_KW,
	"unset":     UNSET_KW,
	"call":      CALL_KW,
	"return":    RETURN_KW,
	"synthetic": SYNTHETIC_KW,
	"error":     ERROR_KW,
	"restart":   RESTART_KW,
	"pass":      PASS_KW,
	"pipe":      PIPE_KW,
	"hash":      HASH_KW,
	"lookup":    LOOKUP_KW,
	"miss":      MISS_KW,
	"hit":       HIT_KW,
	"fetch":     FETCH_KW,
	"deliver":   DELIVER_KW,
	"purge":     PURGE_KW,
	"synth":     SYNTH_KW,
	"abandon":   ABANDON_KW,
	"retry":     RETRY_KW,
	"ok":        OK_KW,
	"fail":      FAIL_KW,
	"new":       NEW_KW,
}

// LookupKeyword returns the token type for a keyword, or ID if not a keyword
func LookupKeyword(ident string) TokenType {
	if tok, ok := Keywords[ident]; ok {
		return tok
	}
	return ID
}

// IsKeyword returns true if the token type represents a keyword
func (t TokenType) IsKeyword() bool {
	return t >= VCL_KW && t <= FAIL_KW
}

// IsLiteral returns true if the token type represents a literal value
func (t TokenType) IsLiteral() bool {
	return t == ID || t == CNUM || t == FNUM || t == CSTR
}

// IsOperator returns true if the token type represents an operator
func (t TokenType) IsOperator() bool {
	return (t >= INC && t <= NOMATCH) ||
		(t >= MULTIPLY && t <= TILDE && t != SEMICOLON && t != COMMA && t != DOT)
}
