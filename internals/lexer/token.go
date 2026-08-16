package lexer

import (
	"unsafe"
)

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

func (t *Token) GetLexme() string {
	return unsafe.String((*byte)(t.Start), t.Length)
}
