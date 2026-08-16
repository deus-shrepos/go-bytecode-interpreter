package lexer_test

import (
	"testing"
	"time"
	"unsafe"

	"go-bytecode-interpreter/internals/lexer"
)

// byteAt safely dereferences a Token.Start pointer as a single byte.
// Token.GetLexme() is not used here: it reinterprets Start as a *string
// via unsafe.Slice, which panics/produces garbage for pointers into a
// []byte source.
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

// TestScanToken_Unimplemented pins down today's actual behavior for
// TokenTypes the scanner does not yet produce: identifiers, number
// literals, and keywords all fall through to errorToken("Unexpected
// Character.") because ScanToken has no branch for letter/digit starts.
// Multi-char operators and string literals were implemented since this
// test was first written and now have their own tests below.
// These should be updated (not just re-asserted) as scanner.go grows.
func TestScanToken_Unimplemented(t *testing.T) {
	cases := []struct {
		name   string // TODO: update once scanner implements this category
		source string
	}{
		{"IDENTIFIER", "abc"},
		{"NUMBER", "123"},
		{"KEYWORD_and", "and"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := append([]byte(tc.source), 0x00)
			s := lexer.NewScanner(source)
			tok := s.ScanToken()

			if tok.Type != lexer.ERROR {
				t.Errorf("Type = %v, want ERROR (unimplemented today)", tok.Type)
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

// TestScanToken_String_Unterminated pins down today's actual behavior when
// a string literal is never closed: string() loops on peek() != '"' &&
// !isAtEnd(), then unconditionally advances once more past the terminator
// byte. There is no dedicated "unterminated string" error path, so this
// still comes back as STRING rather than ERROR. Flagging for the human;
// not asserting this is desirable, only that it's what happens today.
func TestScanToken_String_Unterminated(t *testing.T) {
	source := []byte("\"abc\x00")
	s := lexer.NewScanner(source)
	tok := s.ScanToken()

	if tok.Type != lexer.STRING {
		t.Errorf("Type = %v, want STRING (unterminated strings are not yet an error today)", tok.Type)
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
