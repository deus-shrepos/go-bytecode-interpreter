package lexer

import (
	"bytes"

	"go-bytecode-interpreter/internals/token"
)

type Scanner struct {
	source []byte
	lines  []uint32
	fstack []uint32 // 1 entry per f-string
	tokens []token.Token
	diags  []token.Diagnostics

	start   uint32
	current uint32
}

func newScanner(source []byte) *Scanner {
	return &Scanner{
		source: source,
		lines:  make([]uint32, 1, bytes.Count(source, []byte{'\n'})+1),
		tokens: make([]token.Token, 0, len(source)/8+1),
		diags:  nil,
	}
}

func ScanTokens(source []byte) (tokens []token.Token, line []uint32, diags []token.Diagnostics) {
	s := newScanner(source)
	for {
		kind := s.scanToken()
		s.EmitToken(kind)
		if kind == token.EOF {
			break
		}
	}
	return s.tokens, s.lines, s.diags
}

func (s *Scanner) scanToken() token.Kind {
	// scan whitespace and advance
	s.skipWhiteSpace()
	// scan the current lexeme (at the start of it)
	s.start = s.current
	if s.isAtEnd() {
		if s.insideFstring() {
			s.fstack = s.fstack[:0]
			return s.ReportError("unterminated f-string")
		}
		// the sentinel value to stop the scanner
		return token.EOF
	}
	c := s.advance()
	if c == 'f' && s.peek() == '"' {
		s.advance()
		return s.fstringText(true)
	}
	if isAlpha(c) {
		return s.makeIdentifier()
	}
	if isDigit(c) {
		return s.makeNumber()
	}

	switch c {
	case '(':
		return token.LEFT_PAREN
	case ')':
		return token.RIGHT_PAREN
	case '{':
		if s.insideFstring() {
			s.fstack[len(s.fstack)-1]++
		}
		return token.LEFT_BRACE
	case '}':
		if s.insideFstring() {
			topBrace := s.fstack[len(s.fstack)-1]
			if topBrace == 0 {
				s.fstack = s.fstack[:len(s.fstack)-1]
				// parse rest of the string
				return s.fstringText(false)
			}
			if topBrace > 0 {
				s.fstack[len(s.fstack)-1]--
				return token.RIGHT_BRACE
			}
		}
		return token.RIGHT_BRACE
	case ';':
		return token.SEMICOLON
	case ',':
		return token.COMMA
	case '.':
		return token.DOT
	case '-':
		return token.MINUS
	case '+':
		return token.PLUS
	case '/':
		return token.SLASH
	case '*':
		return token.STAR
	case '!':
		if s.match('=') {
			return token.BANG_EQUAL
		} else {
			return token.BANG
		}
	case '=':
		if s.match('=') {
			return token.EQUAL_EQUAL
		} else {
			return token.EQUAL
		}

	case '<':
		if s.match('=') {
			return token.LESS_EQUAL
		} else {
			return token.LESS
		}
	case '>':
		if s.match('=') {
			return token.GREATER_EQUAL
		} else {
			return token.GREATER
		}
	case '"':
		return s.string()
	default:
		return s.ReportError("Unexpected Token")
	}
}

func (s *Scanner) EmitToken(kind token.Kind) {
	s.tokens = append(s.tokens, token.Token{
		Type:  kind,
		Start: s.start,
		End:   s.current,
	})
}

func (s *Scanner) skipWhiteSpace() {
	for {
		// return the current token and only advance
		// when a whitespace byte is encountered
		c := s.peek()
		switch c {
		case '\t', '\r', ' ', '\n':
			s.advance()
		// case '\n':
		// 	s.advance()
		// 	s.newLineStarts()
		// 	continue
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
func (s *Scanner) string() token.Kind {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\\' {
			s.advance()
			if !s.isAtEnd() {
				s.advance()
			}
			continue
		}

		// if s.peek() == '\n' {
		// 	s.advance()
		// 	continue
		// }
		s.advance()
	}

	if s.isAtEnd() {
		return s.ReportError("Unterminated string")

	}
	s.advance() // need to close the qoute
	return token.STRING
}

func (s *Scanner) fstringText(isFirst bool) token.Kind {
	for !s.isAtEnd() {
		if s.peek() == '\\' {
			s.advance()
			if !s.isAtEnd() {
				s.advance()
			}
			continue
		}
		// if s.peek() == '\n' {
		// 	s.advance()
		// 	s.newLineStarts()
		// 	continue
		// }
		if s.peek() == '{' {
			s.advance()
			s.fstack = append(s.fstack, 0)
			if isFirst {
				return token.F_STRING_START
			} else {
				return token.F_STRING_MID
			}
		}
		if s.peek() == '"' {
			s.advance()
			return token.F_STRING_END
		}
		s.advance()
	}

	// end of the input
	return s.ReportError("unterminated f-stirng")

}

func (s *Scanner) newLineStarts() {
	s.lines = append(s.lines, s.current)
}

func (s *Scanner) makeNumber() token.Kind {
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
	return token.NUMBER
}

func (s *Scanner) makeIdentifier() token.Kind {
	for isAlpha(s.peek()) || isDigit(s.peek()) {
		s.advance()
	}
	return s.identifierType()
}

func (s *Scanner) peek() byte {
	if s.isAtEnd() {
		return 0x00
	}
	return s.source[s.current]
}

func (s *Scanner) peekNext() byte {
	if s.isAtEnd() || int(s.current+1) >= len(s.source) {
		return 0x00
	}
	return s.source[s.current+1]
}

func (s *Scanner) match(b byte) bool {
	if s.isAtEnd() {
		return false
	}
	if s.source[s.current] != b {
		return false
	}
	s.current = s.current + 1
	return true
}

func (s *Scanner) identifierType() token.Kind {
	switch s.source[s.start] {
	case 'a':
		return s.checkKeyword(1, 2, "nd", token.AND)
	case 'c':
		return s.checkKeyword(1, 4, "lass", token.CLASS)
	case 'e':
		return s.checkKeyword(1, 3, "lse", token.ELSE)
	case 'f':
		// the length of the chars should not exceed 3 bytes
		if s.current-s.start >= 3 {
			next := s.source[s.start+1]
			switch next {
			case 'a':
				return s.checkKeyword(2, 3, "lse", token.FALSE)
			case 'u':
				return s.checkKeyword(2, 1, "n", token.FUN)
			case 'o':
				return s.checkKeyword(2, 1, "r", token.FOR)
			}
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
		if s.current-s.start >= 2 {
			next := s.source[s.start+1]
			switch next {
			case 'h':
				return s.checkKeyword(2, 2, "is", token.THIS)
			case 'r':
				return s.checkKeyword(2, 2, "ue", token.TRUE)
			}
		}
	case 'v':
		return s.checkKeyword(1, 2, "ar", token.VAR)
	case 'w':
		return s.checkKeyword(1, 4, "hile", token.WHILE)
	}

	return token.IDENTIFIER
}

func (s *Scanner) checkKeyword(start uint32, length uint32, rest string, tokenType token.Kind) token.Kind {
	// check the length and match the strings
	// we move the s.start to the current start position
	if (s.current-s.start == (start + length)) &&
		bytes.Equal(s.source[s.start+start:s.current], []byte(rest)) {
		return tokenType
	}
	return token.IDENTIFIER
}

func (s *Scanner) isAtEnd() bool {
	return int(s.current) >= len(s.source)
}

func (s *Scanner) advance() byte {
	c := s.source[s.current]
	s.current++
	if c == '\n' {
		s.newLineStarts()
	}
	return c
}

func (s *Scanner) insideFstring() bool {
	return len(s.fstack) > 0
}

func (s *Scanner) ReportError(message string) token.Kind {
	s.diags = append(s.diags, token.Diagnostics{
		Start:   s.start,
		End:     s.current,
		Message: message,
	})
	return token.ERROR
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}
