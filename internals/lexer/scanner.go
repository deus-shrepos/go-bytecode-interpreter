package lexer

import (
	"fmt"
	"strings"
	"unsafe"
)

type Scanner struct {
	start   unsafe.Pointer
	current unsafe.Pointer
	line    int
	source  []byte // arena?
}

func NewScanner(source []byte) *Scanner {
	return &Scanner{
		start:   unsafe.Pointer(&source[0]),
		current: unsafe.Pointer(&source[0]),
		source:  source,
		line:    1,
	}
}

func (s *Scanner) ScanToken() Token {
	// scan whitespace and advance
	s.skipWhiteSpace()
	// scan the current lexeme (at the start of it)
	s.start = s.current
	if s.isAtEnd() {
		// the sentinel value to stop the compiler
		return s.makeToken(EOF)
	}
	c := s.advance()
	if isAlpha(c) {
		return s.makeIdentifier()
	}
	if isDigit(c) {
		return s.makeNumber()
	}

	switch c {
	case '(':
		return s.makeToken(LEFT_PAREN)
	case ')':
		return s.makeToken(RIGHT_PAREN)
	case '{':
		return s.makeToken(LEFT_BRACE)
	case '}':
		return s.makeToken(RIGHT_BRACE)
	case ';':
		return s.makeToken(SEMICOLON)
	case ',':
		return s.makeToken(COMMA)
	case '.':
		return s.makeToken(DOT)
	case '-':
		return s.makeToken(MINUS)
	case '+':
		return s.makeToken(PLUS)
	case '/':
		return s.makeToken(SLASH)
	case '*':
		return s.makeToken(STAR)
	case '!':
		if s.match('=') {
			return s.makeToken(BANG_EQUAL)
		} else {
			return s.makeToken(BANG)
		}
	case '=':
		if s.match('=') {
			return s.makeToken(EQUAL_EQUAL)
		} else {
			return s.makeToken(EQUAL)
		}

	case '<':
		if s.match('=') {
			return s.makeToken(LESS_EQUAL)
		} else {
			return s.makeToken(LESS)
		}
	case '>':
		if s.match('=') {
			return s.makeToken(GREATER_EQUAL)
		} else {
			return s.makeToken(GREATER)
		}
	case '"':
		return s.string()

	default:
		return s.errorToken(fmt.Sprintf("Unexpected Token: %q (%d)", c, c))
	}
}
func (s *Scanner) makeToken(ttype TokenType) Token {
	return Token{
		Type:   ttype,
		Start:  s.start,
		Length: (int)(uintptr(unsafe.Add(s.current, -uintptr(s.start)))), // lexeme[n] - lexeme[0]
		Line:   s.line,
	}
}

func (s *Scanner) skipWhiteSpace() {
	for {
		// return the current token and only advance
		// when a whitespace byte is encountered
		c := s.peek()
		switch c {
		case '\t', '\r', ' ':
			s.advance()
		case '\n':
			s.line++
			s.advance()
		case '/':
			if s.peekNext() == '/' {
				// it's a comment and we consume until we reach newline
				// or we are the end
				for s.peek() != '\n' && !s.isAtEnd() {
					s.advance()
				}
			} else {
				// we return back to scanToken() and advance as normal
				return
			}
		default:
			return
		}
	}
}
func (s *Scanner) string() Token {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}
		s.advance()
	}

	if s.isAtEnd() {
		return s.errorToken("Unterminated string")
	}
	s.advance() // need to close the qoute
	return s.makeToken(STRING)
}

func (s *Scanner) makeNumber() Token {
	// find the digit and keep advancing
	for isDigit(s.peek()) {
		s.advance()
	}

	// scan afer the "." part of the digit, the fractional part
	if s.peek() == '.' && isDigit(s.peekNext()) {
		s.advance() // consume "."
		for isDigit(s.peek()) {
			s.advance()
		}
	}
	return s.makeToken(NUMBER)
}

func (s *Scanner) makeIdentifier() Token {
	for isAlpha(s.peek()) || isDigit(s.peek()) {
		s.advance()
	}
	return s.makeToken(s.identifierType())
}

func (s *Scanner) peek() byte {
	return *(*byte)(s.current)
}

func (s *Scanner) peekNext() byte {
	if s.isAtEnd() {
		return 0x00
	}
	return *(*byte)(unsafe.Add(s.current, 1))
}

func (s *Scanner) match(b byte) bool {
	if s.isAtEnd() {
		return false
	}
	if *(*byte)(unsafe.Pointer(s.current)) != b {
		return false
	}
	s.current = unsafe.Add(s.current, 1)
	return true
}

func (s *Scanner) identifierType() TokenType {
	switch *(*byte)(s.start) {
	case 'a':
		return s.checkKeyword(1, 2, "nd", AND)
	case 'c':
		return s.checkKeyword(1, 4, "lass", CLASS)
	case 'e':
		return s.checkKeyword(1, 3, "lse", ELSE)
	case 'f':
		next := *(*byte)(unsafe.Add(s.start, 1))
		switch next {
		case 'a':
			return s.checkKeyword(2, 3, "lse", FALSE)
		case 'u':
			return s.checkKeyword(2, 1, "n", FUN)
		case 'o':
			return s.checkKeyword(2, 1, "r", FOR)
		}
	case 'i':
		return s.checkKeyword(1, 1, "f", IF)
	case 'n':
		return s.checkKeyword(1, 2, "il", NIL)
	case 'o':
		return s.checkKeyword(1, 1, "r", OR)
	case 'p':
		return s.checkKeyword(1, 4, "rint", PRINT)
	case 'r':
		return s.checkKeyword(1, 5, "eturn", RETURN)
	case 's':
		return s.checkKeyword(1, 4, "uper", SUPER)
	case 't':
		next := *(*byte)(unsafe.Add(s.start, 1))
		switch next {
		case 'h':
			return s.checkKeyword(2, 2, "is", THIS)
		case 'r':
			return s.checkKeyword(2, 2, "ue", TRUE)
		}
	case 'v':
		return s.checkKeyword(1, 2, "ar", VAR)
	case 'w':
		return s.checkKeyword(1, 4, "hile", WHILE)
	}

	return IDENTIFIER
}

func (s *Scanner) checkKeyword(start int, length int, rest string, tokenType TokenType) TokenType {
	// check the length and match the strings
	// we move the s.start to the current start position
	if (calcPtrDiff(s.start, s.current) == (start + length)) &&
		(MatchString(unsafe.Add(s.start, start), rest, length) == 0) {
		return tokenType
	}
	return IDENTIFIER
}

func (s *Scanner) isAtEnd() bool {
	return *(*byte)(s.current) == 0x00
}

func (s *Scanner) advance() byte {
	s.current = unsafe.Add(s.current, 1)
	return *(*byte)(unsafe.Add(s.current, -1))
}

func (s *Scanner) errorToken(message string) Token {
	byteString := []byte(message)
	return Token{
		Type:   ERROR,
		Start:  unsafe.Pointer(&byteString[0]),
		Length: len(message),
		Line:   s.line,
	}
}

func MatchString(basePtr unsafe.Pointer, rest string, length int) int {
	return strings.Compare(unsafe.String((*byte)(basePtr), length), rest)
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func calcPtrDiff(a unsafe.Pointer, b unsafe.Pointer) int {
	return (int)(uintptr(b) - uintptr(a))
}
