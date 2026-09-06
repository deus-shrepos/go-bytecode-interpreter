package lexer

import (
	"fmt"
	"go-bytecode-interpreter/internals/token"
	"strings"
	"unsafe"
)

type Scanner struct {
	line       int
	interpMode bool
	start      unsafe.Pointer
	current    unsafe.Pointer
	source     []byte // arena?
}

func NewScanner(source []byte) *Scanner {
	return &Scanner{
		start:      unsafe.Pointer(&source[0]),
		current:    unsafe.Pointer(&source[0]),
		source:     source,
		line:       1,
		interpMode: false,
	}
}

func (s *Scanner) ScanToken() token.Token {
	// scan whitespace and advance
	s.skipWhiteSpace()
	// scan the current lexeme (at the start of it)
	s.start = s.current
	if s.isAtEnd() {
		// the sentinel value to stop the compiler
		return s.makeToken(token.EOF)
	}
	c := s.advance()
	if c == 'f' && s.peek() == '"' {
		s.advance()
		return s.fstringStart()
	}
	if isAlpha(c) {
		return s.makeIdentifier()
	}
	if isDigit(c) {
		return s.makeNumber()
	}

	switch c {
	case '(':
		return s.makeToken(token.LEFT_PAREN)
	case ')':
		return s.makeToken(token.RIGHT_PAREN)
	case '{':
		return s.makeToken(token.LEFT_BRACE)
	case '}':
		if s.interpMode == true {
			s.interpMode = false
			if s.fstringScan() == '"' {
				s.advance()
				return s.makeToken(token.F_STRING_END)
			}
			if s.fstringScan() == '{' {
				s.interpMode = true
				s.advance()
				return s.makeToken(token.F_STRING_MID)
			}
		}
		return s.makeToken(token.RIGHT_BRACE)
	case ';':
		return s.makeToken(token.SEMICOLON)
	case ',':
		return s.makeToken(token.COMMA)
	case '.':
		return s.makeToken(token.DOT)
	case '-':
		return s.makeToken(token.MINUS)
	case '+':
		return s.makeToken(token.PLUS)
	case '/':
		return s.makeToken(token.SLASH)
	case '*':
		return s.makeToken(token.STAR)
	case '!':
		if s.match('=') {
			return s.makeToken(token.BANG_EQUAL)
		} else {
			return s.makeToken(token.BANG)
		}
	case '=':
		if s.match('=') {
			return s.makeToken(token.EQUAL_EQUAL)
		} else {
			return s.makeToken(token.EQUAL)
		}

	case '<':
		if s.match('=') {
			return s.makeToken(token.LESS_EQUAL)
		} else {
			return s.makeToken(token.LESS)
		}
	case '>':
		if s.match('=') {
			return s.makeToken(token.GREATER_EQUAL)
		} else {
			return s.makeToken(token.GREATER)
		}
	case '"':
		return s.string()

	default:
		return s.errorToken(fmt.Sprintf("Unexpected Token: %q (%d)", c, c))
	}
}

func (s *Scanner) makeToken(ttype token.Kind) token.Token {
	return token.Token{
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
func (s *Scanner) string() token.Token {
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
	return s.makeToken(token.STRING)
}

func (s *Scanner) fstringStart() token.Token {
	if s.fstringScan() == '{' {
		s.interpMode = true
		s.advance()
		return s.makeToken(token.F_STRING_START)
	}

	// in case we don't find any "{}"
	// we will just parse it as a string
	s.advance()
	return s.makeToken(token.F_STRING_END)
}

func (s *Scanner) fstringScan() byte {
	for (s.peek() != '"' && s.peek() != '{') && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}
		s.advance()
	}
	return s.peek()
}

func (s *Scanner) makeNumber() token.Token {
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
	return s.makeToken(token.NUMBER)
}

func (s *Scanner) makeIdentifier() token.Token {
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

func (s *Scanner) identifierType() token.Kind {
	switch *(*byte)(s.start) {
	case 'a':
		return s.checkKeyword(1, 2, "nd", token.AND)
	case 'c':
		return s.checkKeyword(1, 4, "lass", token.CLASS)
	case 'e':
		return s.checkKeyword(1, 3, "lse", token.ELSE)
	case 'f':
		next := *(*byte)(unsafe.Add(s.start, 1))
		switch next {
		case 'a':
			return s.checkKeyword(2, 3, "lse", token.FALSE)
		case 'u':
			return s.checkKeyword(2, 1, "n", token.FUN)
		case 'o':
			return s.checkKeyword(2, 1, "r", token.FOR)
		}
	case 'i':
		return s.checkKeyword(1, 1, "f", token.IF)
	case 'n':
		return s.checkKeyword(1, 2, "il", token.NIL)
	case 'o':
		return s.checkKeyword(1, 1, "r", token.OR)
	case 'p':
		return s.checkKeyword(1, 4, "rint", token.PRINT)
	case 'r':
		return s.checkKeyword(1, 5, "eturn", token.RETURN)
	case 's':
		return s.checkKeyword(1, 4, "uper", token.SUPER)
	case 't':
		next := *(*byte)(unsafe.Add(s.start, 1))
		switch next {
		case 'h':
			return s.checkKeyword(2, 2, "is", token.THIS)
		case 'r':
			return s.checkKeyword(2, 2, "ue", token.TRUE)
		}
	case 'v':
		return s.checkKeyword(1, 2, "ar", token.VAR)
	case 'w':
		return s.checkKeyword(1, 4, "hile", token.WHILE)
	}

	return token.IDENTIFIER
}

func (s *Scanner) checkKeyword(start int, length int, rest string, tokenType token.Kind) token.Kind {
	// check the length and match the strings
	// we move the s.start to the current start position
	if (calcPtrDiff(s.start, s.current) == (start + length)) &&
		(MatchString(unsafe.Add(s.start, start), rest, length) == 0) {
		return tokenType
	}
	return token.IDENTIFIER
}

func (s *Scanner) isAtEnd() bool {
	return *(*byte)(s.current) == 0x00
}

func (s *Scanner) advance() byte {
	s.current = unsafe.Add(s.current, 1)
	return *(*byte)(unsafe.Add(s.current, -1))
}

func (s *Scanner) errorToken(message string) token.Token {
	byteString := []byte(message)
	return token.Token{
		Type:   token.ERROR,
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
