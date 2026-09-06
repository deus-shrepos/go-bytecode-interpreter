package token

import (
	"fmt"
	"os"
	"unsafe"
)

//go:generate stringer -type=TokenType
type Kind uint8

const (

	// Single character tokens
	LEFT_PAREN Kind = iota
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	DOT
	MINUS
	PLUS
	SEMICOLON
	SLASH
	STAR

	// One or two characters tokens
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL

	// Literals
	IDENTIFIER
	STRING
	NUMBER

	// Keywords
	AND
	CLASS
	ELSE
	FALSE
	FOR
	FUN
	IF
	NIL
	OR
	PRINT
	RETURN
	SUPER
	THIS
	TRUE
	VAR
	WHILE

	// F-string
	F_STRING_START
	F_STRING_MID
	F_STRING_END

	ERROR
	EOF
)

type Token struct {
	Start  unsafe.Pointer
	Length int
	Line   int
	Type   Kind
}

func NewToken(ttype Kind, start unsafe.Pointer, length int, line int) Token {
	return Token{
		Type:   ttype,
		Start:  start,
		Length: length,
		Line:   line,
	}
}

func (t Token) Kind() Kind {
	return t.Type
}
func (t Token) Lexeme() string {
	return unsafe.String((*byte)(t.Start), t.Length)
}

func (t *Token) Print() {
	fmt.Fprintf(os.Stdout, "Token[Type=%v, Start=%p, length=%d, line=%d]",
		t.Type, *(*int)(t.Start), t.Length, t.Line)
}
