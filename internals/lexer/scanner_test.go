package lexer_test

import (
	"testing"
	"time"
	"unsafe"

	"go-bytecode-interpreter/internals/lexer"
)

// byteAt safely dereferences a Token.Start pointer as a single byte.
func byteAt(p unsafe.Pointer) byte {
	return *(*byte)(p)
}

func TestScanToken_SingleCharTokens(t *testing.T) {
	cases := []struct {
		name string
		ch   byte
		want lexer.TokenType
	}{
		{"LEFT_PAREN", '(', lexer.LEFT_PAREN},
		{"RIGHT_PAREN", ')', lexer.RIGHT_PAREN},
		{"LEFT_BRACE", '{', lexer.LEFT_BRACE},
		{"RIGHT_BRACE", '}', lexer.RIGHT_BRACE},
		{"COMMA", ',', lexer.COMMA},
		{"DOT", '.', lexer.DOT},
		{"MINUS", '-', lexer.MINUS},
		{"PLUS", '+', lexer.PLUS},
		{"SEMICOLON", ';', lexer.SEMICOLON},
		{"SLASH", '/', lexer.SLASH},
		{"STAR", '*', lexer.STAR},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := []byte{tc.ch, 0x00}
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != tc.want {
				t.Errorf("Type = %v, want %v", tok.Type, tc.want)
			}
			if tok.Length != 1 {
				t.Errorf("Length = %d, want 1", tok.Length)
			}
			if got := byteAt(tok.Start); got != tc.ch {
				t.Errorf("byte at Start = %q, want %q", got, tc.ch)
			}
		})
	}
}

func TestScanToken_MultiTokenSequence(t *testing.T) {
	// Mirrors examples/scanner_text.glox: "(+@+)"
	source := []byte("(+@+)\x00")
	s := lexer.NewScanner(source)

	want := []lexer.TokenType{
		lexer.LEFT_PAREN,
		lexer.PLUS,
		lexer.ERROR,
		lexer.PLUS,
		lexer.RIGHT_PAREN,
	}

	for i, wantType := range want {
		tok := s.ScanToken()
		if tok.Type != wantType {
			t.Fatalf("token %d: Type = %v, want %v", i, tok.Type, wantType)
		}
	}

	if tok := s.ScanToken(); tok.Type != lexer.EOF {
		t.Errorf("final token Type = %v, want EOF", tok.Type)
	}
}

func TestScanToken_EOFOnEmptySource(t *testing.T) {
	source := []byte{0x00}
	s := lexer.NewScanner(source)

	tok := s.ScanToken()
	if tok.Type != lexer.EOF {
		t.Errorf("Type = %v, want EOF", tok.Type)
	}
}

// TestScanToken_Identifier covers identifier scanning: alpha/underscore
// start, continuing over alpha/digit/underscore per isAlpha/isDigit in
// scanner.go.
func TestScanToken_Identifier(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantLength int
	}{
		{"simple", "abc", 3},
		{"single_letter", "x", 1},
		{"leading_underscore", "_foo", 4},
		{"trailing_underscore", "foo_", 4},
		{"internal_underscore", "foo_bar", 7},
		{"trailing_digits", "foo123", 6},
		{"all_underscores", "___", 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.IDENTIFIER {
				t.Fatalf("Type = %v, want IDENTIFIER", tok.Type)
			}
			if tok.Length != tc.wantLength {
				t.Errorf("Length = %d, want %d", tok.Length, tc.wantLength)
			}
			if got := tok.GetLexme(); got != tc.source {
				t.Errorf("GetLexme() = %q, want %q", got, tc.source)
			}
		})
	}
}

// TestScanToken_Number covers integer and fractional literals, plus the
// "trailing dot with no following digit" edge case: makeNumber only
// consumes the '.' when peekNext() is a digit, so "123." must scan as
// NUMBER("123") followed by a separate DOT token, not a malformed number.
func TestScanToken_Number(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantLength int
	}{
		{"integer", "123", 3},
		{"single_digit", "0", 1},
		{"fractional", "123.456", 7},
		{"fractional_single_digit", "1.5", 3},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.NUMBER {
				t.Fatalf("Type = %v, want NUMBER", tok.Type)
			}
			if tok.Length != tc.wantLength {
				t.Errorf("Length = %d, want %d", tok.Length, tc.wantLength)
			}
		})
	}
}

func TestScanToken_Number_TrailingDotNotConsumed(t *testing.T) {
	source := []byte("123.\x00")
	s := lexer.NewScanner(source)

	numTok := s.ScanToken()
	if numTok.Type != lexer.NUMBER {
		t.Fatalf("first token Type = %v, want NUMBER", numTok.Type)
	}
	if numTok.Length != 3 {
		t.Errorf("first token Length = %d, want 3 (dot not consumed without a following digit)", numTok.Length)
	}

	dotTok := s.ScanToken()
	if dotTok.Type != lexer.DOT {
		t.Errorf("second token Type = %v, want DOT", dotTok.Type)
	}
}

// TestScanToken_Number_MultipleDots mirrors clox/lox behavior: "1.2.3" is
// NUMBER("1.2"), DOT, NUMBER("3") — a number literal has at most one
// fractional part, the second '.' starts a fresh scan.
func TestScanToken_Number_MultipleDots(t *testing.T) {
	source := []byte("1.2.3\x00")
	s := lexer.NewScanner(source)

	want := []struct {
		ttype  lexer.TokenType
		length int
	}{
		{lexer.NUMBER, 3}, // "1.2"
		{lexer.DOT, 1},
		{lexer.NUMBER, 1}, // "3"
	}

	for i, w := range want {
		tok := s.ScanToken()
		if tok.Type != w.ttype {
			t.Fatalf("token %d: Type = %v, want %v", i, tok.Type, w.ttype)
		}
		if tok.Length != w.length {
			t.Errorf("token %d: Length = %d, want %d", i, tok.Length, w.length)
		}
	}
}

// TestScanToken_Keywords asserts every reserved word in the language
// (token.go's Keywords block) scans to its dedicated TokenType rather than
// IDENTIFIER. Table mirrors clox's keyword set exactly.
func TestScanToken_Keywords(t *testing.T) {
	cases := []struct {
		source string
		want   lexer.TokenType
	}{
		{"and", lexer.AND},
		{"class", lexer.CLASS},
		{"else", lexer.ELSE},
		{"false", lexer.FALSE},
		{"for", lexer.FOR},
		{"fun", lexer.FUN},
		{"if", lexer.IF},
		{"nil", lexer.NIL},
		{"or", lexer.OR},
		{"print", lexer.PRINT},
		{"return", lexer.RETURN},
		{"super", lexer.SUPER},
		{"this", lexer.THIS},
		{"true", lexer.TRUE},
		{"var", lexer.VAR},
		{"while", lexer.WHILE},
	}

	for _, tc := range cases {
		t.Run(tc.source, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != tc.want {
				t.Errorf("Type = %v, want %v", tok.Type, tc.want)
			}
			if tok.Length != len(tc.source) {
				t.Errorf("Length = %d, want %d", tok.Length, len(tc.source))
			}
		})
	}
}

// TestScanToken_KeywordLikeIdentifiers is the adversarial counterpart to
// TestScanToken_Keywords: identifiers that share a keyword's prefix but are
// longer, or that stop short of a keyword's full spelling, must scan as
// IDENTIFIER. checkKeyword's job (scanner.go) is exactly this
// discrimination — it should reject on either a length mismatch or a byte
// mismatch in the remainder, meaning BOTH conditions must hold for a
// keyword match, not just one.
func TestScanToken_KeywordLikeIdentifiers(t *testing.T) {
	cases := []string{
		// longer than the keyword, sharing its full prefix
		"andor", "classroom", "elsewhere", "falsely", "forward",
		"funny", "iffy", "nile", "orange", "printer", "returning",
		"superb", "thistle", "truest", "vary", "whiley",
		// shorter than the keyword, a strict prefix of it
		"an", "cla", "el", "fal", "fo", "fu", "ni", "pri", "ret",
		"sup", "th", "tr", "va", "whil",
		// single letter, just the switch discriminator itself
		"a", "c", "e", "f", "i", "n", "o", "p", "r", "s", "t", "v", "w",
		// keyword text with a trailing digit/underscore (still one identifier)
		"class1", "var_",
	}

	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			source := append([]byte(src), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.IDENTIFIER {
				t.Errorf("source %q: Type = %v, want IDENTIFIER", src, tok.Type)
			}
			if tok.Length != len(src) {
				t.Errorf("source %q: Length = %d, want %d", src, tok.Length, len(src))
			}
		})
	}
}

// TestScanToken_KeywordCaseSensitivity asserts keyword matching is
// lowercase-only, per the literal switch cases in identifierType.
func TestScanToken_KeywordCaseSensitivity(t *testing.T) {
	cases := []string{"AND", "And", "TRUE", "Nil"}

	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			source := append([]byte(src), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.IDENTIFIER {
				t.Errorf("source %q: Type = %v, want IDENTIFIER (keywords are lowercase-only)", src, tok.Type)
			}
		})
	}
}

func TestScanToken_TwoCharOperators(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantType   lexer.TokenType
		wantLength int
	}{
		{"BANG", "!", lexer.BANG, 1},
		{"BANG_EQUAL", "!=", lexer.BANG_EQUAL, 2},
		{"EQUAL", "=", lexer.EQUAL, 1},
		{"EQUAL_EQUAL", "==", lexer.EQUAL_EQUAL, 2},
		{"LESS", "<", lexer.LESS, 1},
		{"LESS_EQUAL", "<=", lexer.LESS_EQUAL, 2},
		{"GREATER", ">", lexer.GREATER, 1},
		{"GREATER_EQUAL", ">=", lexer.GREATER_EQUAL, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != tc.wantType {
				t.Errorf("Type = %v, want %v", tok.Type, tc.wantType)
			}
			if tok.Length != tc.wantLength {
				t.Errorf("Length = %d, want %d", tok.Length, tc.wantLength)
			}
		})
	}
}

func TestScanToken_String(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantLength int
	}{
		{"basic", "\"abc\"", 5},
		{"empty", "\"\"", 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.STRING {
				t.Errorf("Type = %v, want STRING", tok.Type)
			}
			if tok.Length != tc.wantLength {
				t.Errorf("Length = %d, want %d", tok.Length, tc.wantLength)
			}
			if got := byteAt(tok.Start); got != '"' {
				t.Errorf("byte at Start = %q, want opening quote", got)
			}
		})
	}
}

// TestScanToken_String_Unterminated asserts a string with no closing quote
// is reported as ERROR: string() now checks isAtEnd() before the closing
// advance() and returns errorToken in that case.
func TestScanToken_String_Unterminated(t *testing.T) {
	source := []byte("\"abc\x00")
	s := lexer.NewScanner(source)
	tok := s.ScanToken()

	if tok.Type != lexer.ERROR {
		t.Errorf("Type = %v, want ERROR (unterminated string)", tok.Type)
	}
}

// TestScanToken_String_UnterminatedAcrossNewlines asserts the same for a
// multi-line unterminated string, exercising the line++ branch inside the
// same unterminated scan.
func TestScanToken_String_UnterminatedAcrossNewlines(t *testing.T) {
	source := []byte("\"abc\ndef\x00")
	s := lexer.NewScanner(source)
	tok := s.ScanToken()

	if tok.Type != lexer.ERROR {
		t.Errorf("Type = %v, want ERROR (unterminated string)", tok.Type)
	}
}

// TestScanToken_String_MultilineTerminated asserts a closed string spanning
// multiple lines scans as STRING. makeToken() stamps Line from s.line at the
// point it's called — after the whole lexeme, including internal newlines,
// has been consumed — so the string token's Line reflects its closing
// quote's line, not its opening quote's line. Matches clox: line tracking
// is a side effect of advance(), not a snapshot taken at token start.
func TestScanToken_String_MultilineTerminated(t *testing.T) {
	source := []byte("\"line1\nline2\"\n+\x00")
	s := lexer.NewScanner(source)

	strTok := s.ScanToken()
	if strTok.Type != lexer.STRING {
		t.Fatalf("Type = %v, want STRING", strTok.Type)
	}
	if strTok.Line != 2 {
		t.Errorf("string token Line = %d, want 2 (line of the closing quote, not the opening one)", strTok.Line)
	}

	plusTok := s.ScanToken()
	if plusTok.Type != lexer.PLUS {
		t.Fatalf("Type = %v, want PLUS", plusTok.Type)
	}
	if plusTok.Line != 3 {
		t.Errorf("plus token Line = %d, want 3", plusTok.Line)
	}
}

// scanTokenWithTimeout calls s.ScanToken() on a goroutine and fails the test
// (instead of hanging the whole `go test` run) if it doesn't return in time.
// Needed because skipWhiteSpace's `case '\t':` and `case '\r':` bodies are
// empty in scanner.go today — an empty Go case does NOT fall through to the
// next case (unlike C), so it just exits the switch without calling
// advance(). The enclosing `for { ... }` in skipWhiteSpace then re-peeks the
// same byte forever: scanning a literal tab or '\r' spins CPU indefinitely.
func scanTokenWithTimeout(t *testing.T, s *lexer.Scanner) lexer.Token {
	t.Helper()
	done := make(chan lexer.Token, 1)
	go func() { done <- s.ScanToken() }()
	select {
	case tok := <-done:
		return tok
	case <-time.After(2 * time.Second):
		t.Fatalf("ScanToken did not return within 2s — skipWhiteSpace is likely stuck in an infinite loop on this input (see the empty '\\t'/'\\r' cases in scanner.go)")
		return lexer.Token{}
	}
}

// TestScanToken_SkipWhitespace asserts that runs of whitespace (not just a
// single whitespace byte) are fully skipped between tokens. Uses
// scanTokenWithTimeout because tab/'\r' input is known to hang the scanner
// today (see that helper's comment) — this test documents the hang as a
// failure rather than letting it block the whole suite.
func TestScanToken_SkipWhitespace(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"spaces", "+  +\x00"},
		{"tabs", "+\t\t+\x00"},
		{"carriage_return", "+\r\r+\x00"},
		{"mixed_with_newline", "+ \n +\x00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := lexer.NewScanner([]byte(tc.source))

			first := scanTokenWithTimeout(t, s)
			if first.Type != lexer.PLUS {
				t.Fatalf("first token Type = %v, want PLUS", first.Type)
			}
			second := scanTokenWithTimeout(t, s)
			if second.Type != lexer.PLUS {
				t.Fatalf("second token Type = %v, want PLUS (whitespace run not fully skipped)", second.Type)
			}
		})
	}
}

func TestScanToken_SkipComment(t *testing.T) {
	source := []byte("// a comment\n+\x00")
	s := lexer.NewScanner(source)

	tok := s.ScanToken()
	if tok.Type != lexer.PLUS {
		t.Errorf("Type = %v, want PLUS (comment not fully skipped)", tok.Type)
	}
}

// TestScanToken_LineTracking asserts that each token's Line field reflects
// the source line it was scanned on, for both ordinary tokens (makeToken)
// and error tokens (errorToken).
func TestScanToken_LineTracking(t *testing.T) {
	source := []byte("+\n+\n+\x00")
	s := lexer.NewScanner(source)

	wantLines := []int{1, 2, 3}
	for i, wantLine := range wantLines {
		tok := s.ScanToken()
		if tok.Type != lexer.PLUS {
			t.Fatalf("token %d: Type = %v, want PLUS", i, tok.Type)
		}
		if tok.Line != wantLine {
			t.Errorf("token %d: Line = %d, want %d", i, tok.Line, wantLine)
		}
	}

	errSource := []byte("\n\n@\x00")
	errScanner := lexer.NewScanner(errSource)
	errTok := errScanner.ScanToken()
	if errTok.Type != lexer.ERROR {
		t.Fatalf("Type = %v, want ERROR", errTok.Type)
	}
	if errTok.Line != 3 {
		t.Errorf("errorToken Line = %d, want 3", errTok.Line)
	}
}
