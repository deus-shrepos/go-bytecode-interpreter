package lexer

type Scanner struct {
	start   byte
	current byte
	line    int
	source  []byte
}

func NewScanner(source []byte) *Scanner {
	return &Scanner{
		start:   source[0],
		current: source[0],
		source:  source,
		line:    0,
	}
}

func (s *Scanner) ScanToken() Token {
	s.start = s.current
	if s.isAtEnd() {
		return s.makeToken(EOF)
	}
	return Token{}
}

func (s *Scanner) makeToken(ttype TokenType) Token {
	token := Token{
		ttype:  ttype,
		start:  s.start,
		length: int(s.start - s.current),
	}
	return token
}

func (s *Scanner) isAtEnd() bool {
	return s.current == 0x00
}
