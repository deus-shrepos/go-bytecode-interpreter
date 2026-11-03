package token

import (
	"unsafe"
)

//go:generate stringer -type=Type
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

// Token is a span that covers a specific part of the source.
// It holds the Start <= End <= len(src) invariant, and EOF
// is the single last token, with Start == End == len(src)
// All of them counted in bytes
type Token struct {
	Start uint32
	End   uint32
	Type  Kind
}

// Diagnostics It will carry the error information so printing
// does not need the source
type Diagnostics struct {
	Start   uint32
	End     uint32
	Message string
}

func NewToken(kind Kind) Token {
	return Token{
		Type: kind,
	}
}

func (t Token) Kind() Kind {
	return t.Type
}

func (t Token) Lexeme(source []byte) string {
	if t.Len() == 0 {
		return ""
	}
	return unsafe.String(&source[t.Start], t.Len())
}
func (t Token) Len() int { return int(t.End - t.Start) }

// FindLineColumn Finds the line and column position using binary search
func FindLineColumn(lines []uint32, offset uint32) (line, col int) {
	low := 0
	high := len(lines)
	for low < high {
		mid := low + (high-low)/2
		if lines[mid] > offset {
			high = mid
		} else {
			low = mid + 1
		}
	}
	index := low - 1
	line = index + 1
	col = int(offset - lines[index])
	return line, col
}
