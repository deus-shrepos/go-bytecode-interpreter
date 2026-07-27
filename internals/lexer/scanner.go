package lexer

import "unsafe"

type Scanner struct {
	start   unsafe.Pointer
	current unsafe.Pointer
	line    int
	source  []byte
}

func NewScanner(source []byte) *Scanner {
	return &Scanner{
		start:   unsafe.Pointer(&source[0]),
		current: unsafe.Pointer(&source[0]),
		source:  source,
		line:    0,
	}
}

func (s *Scanner) ScanToken() Token {
	// scan the current lexeme (at the start of it)
	s.start = s.current
	if s.isAtEnd() {
		// the sentinel value to stop the compiler
		return s.makeToken(EOF)
	}
	c := s.advance()

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
	}
	return s.errorToken("Unexpected Character.")
}

func (s *Scanner) makeToken(ttype TokenType) Token {
	return Token{
		ttype:  ttype,
		start:  s.start,
		length: *(*int)(unsafe.Add(s.start, -uintptr(s.current))), // lexeme[n] - lexeme[0]
	}
}

func (s *Scanner) isAtEnd() bool {
	return *(*byte)(s.current) == 0x00
}

func (s *Scanner) advance() byte {
	s.current = unsafe.Add(s.current, 1)
	return *(*byte)(unsafe.Add(s.current, -1))
}

func (s *Scanner) errorToken(message string) Token {
	return Token{
		ttype:  ERROR,
		start:  unsafe.Pointer(&message),
		length: len(message),
		line:   s.line,
	}
}
