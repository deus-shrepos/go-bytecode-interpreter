package lexer

import (
	"fmt"
	"unsafe"
)

//go:generate stringer -type=TokenType
type TokenType uint8

const (

	// Single character tokens
	LEFT_PAREN TokenType = iota
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
	Type   TokenType
	Start  unsafe.Pointer
	Length int
	Line   int
}

func NewToken(ttype TokenType, start unsafe.Pointer, length int, line int) Token {
	return Token{
		Type:   ttype,
		Start:  start,
		Length: length,
		Line:   line,
	}
}

func (t *Token) Lexeme() string {
	return unsafe.String((*byte)(t.Start), t.Length)
}

func (t *Token) String() string {
	return fmt.Sprintf("Token[type=%v, start=%p, length=%d, line=%d]", t.Type, t.Start, t.Length, t.Line)

}
