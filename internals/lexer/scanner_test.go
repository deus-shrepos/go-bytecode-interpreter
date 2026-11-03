package lexer_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/token"
)

// ---------------------------------------------------------------------------
// Helpers
//
// Every test goes through scan(), so it is the only place that knows the
// ScanTokens signature. If the signature changes, fix it here.
// ---------------------------------------------------------------------------

// tk is one expected token: its kind and its exact text in the source.
type tk struct {
	kind   token.Kind
	lexeme string
}

func (t tk) String() string { return fmt.Sprintf("%v %q", t.kind, t.lexeme) }

type result struct {
	src        []byte
	toks       []token.Token
	lineStarts []uint32
	diags      []token.Diagnostics
}

// scan runs the scanner with a timeout, so a scanner stuck in a loop fails
// the test instead of hanging the whole `go test` run.
func scan(t *testing.T, src string) result {
	t.Helper()
	done := make(chan result, 1)
	go func() {
		b := []byte(src) // note: no trailing 0x00 sentinel
		toks, lines, diags := lexer.ScanTokens(b)
		done <- result{b, toks, lines, diags}
	}()
	select {
	case r := <-done:
		return r
	case <-time.After(2 * time.Second):
		t.Fatalf("ScanTokens(%q) did not return within 2s (infinite loop?)", src)
		return result{}
	}
}

// got converts scanned tokens into (kind, lexeme) pairs for comparison.
func (r result) got() []tk {
	out := make([]tk, len(r.toks))
	for i, tok := range r.toks {
		out[i] = tk{tok.Type, tok.Lexeme(r.src)}
	}
	return out
}

// expectTokens scans src and checks the full token stream, EOF included.
func expectTokens(t *testing.T, src string, want ...tk) result {
	t.Helper()
	r := scan(t, src)
	want = append(want, tk{token.EOF, ""})
	got := r.got()

	if len(got) != len(want) {
		t.Fatalf("source %q:\n got  %d tokens: %v\n want %d tokens: %v",
			src, len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("source %q: token %d = %v, want %v", src, i, got[i], want[i])
		}
	}
	return r
}

// lineOf returns the 1-based line of token i.
func (r result) lineOf(i int) int {
	line, _ := token.FindLineColumn(r.lineStarts, r.toks[i].Start)
	return line
}

// ---------------------------------------------------------------------------
// Invariants that must hold for ANY input
// ---------------------------------------------------------------------------

// Sources covering every scanner path, valid and invalid.
var invariantSources = []string{
	"",
	"   ",
	"(+@+)",
	"let x = 1.5;\nprint x;",
	`"abc`,
	"\"a\nb\"",
	`f"a {b} c"`,
	`f"abc`,
	"// only a comment",
	"a\x00b",
	"\"a\n",      // unterminated, newline is the last byte
	"f\"a\n",     // same for an f-string
	`"abc\`,      // backslash is the last byte
	`f"{x}" }`,   // stray '}' after a closed f-string
	"\"\n\n\"",   // consecutive newlines inside a string
	`f"{ "}" }"`, // string inside a hole
}

func TestInvariants(t *testing.T) {
	for _, src := range invariantSources {
		t.Run(fmt.Sprintf("%q", src), func(t *testing.T) {
			r := scan(t, src)
			n := uint32(len(src))

			// Exactly one EOF, last, empty, at len(src).
			if len(r.toks) == 0 {
				t.Fatal("no tokens; want at least EOF")
			}
			last := r.toks[len(r.toks)-1]
			if last.Type != token.EOF || last.Start != n || last.End != n {
				t.Errorf("last token = %v [%d,%d), want EOF [%d,%d)",
					last.Type, last.Start, last.End, n, n)
			}

			prevEnd := uint32(0)
			errorTokens := 0
			for i, tok := range r.toks {
				if i < len(r.toks)-1 && tok.Type == token.EOF {
					t.Errorf("token %d is EOF but is not last", i)
				}
				if !(tok.Start <= tok.End && tok.End <= n) {
					t.Errorf("token %d span [%d,%d) breaks Start <= End <= len(src)=%d",
						i, tok.Start, tok.End, n)
				}
				if tok.Start < prevEnd {
					t.Errorf("token %d starts at %d, overlapping the previous token (ends %d)",
						i, tok.Start, prevEnd)
				}
				prevEnd = tok.End
				if tok.Type == token.ERROR {
					errorTokens++
				}
			}

			// Every ERROR token has exactly one diagnostic, and vice versa.
			if errorTokens != len(r.diags) {
				t.Errorf("%d ERROR tokens but %d diagnostics", errorTokens, len(r.diags))
			}

			// lineStarts always begins with 0.
			if len(r.lineStarts) == 0 || r.lineStarts[0] != 0 {
				t.Errorf("lineStarts = %v, want it to start with 0", r.lineStarts)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Cursor / end-of-input
// ---------------------------------------------------------------------------

func TestEmptySource(t *testing.T) {
	expectTokens(t, "")
}

func TestWhitespaceOnly(t *testing.T) {
	expectTokens(t, " \t\r\n ")
}

// The source has no 0x00 sentinel any more: the end is len(src).
// A single-byte source must not read past the slice.
func TestNoSentinelNeeded(t *testing.T) {
	expectTokens(t, "(", tk{token.LEFT_PAREN, "("})
}

// A NUL byte in the source is just a bad byte, not the end of input.
// Scanning must continue past it.
func TestEmbeddedNulIsNotEnd(t *testing.T) {
	expectTokens(t, "a\x00b",
		tk{token.IDENTIFIER, "a"},
		tk{token.ERROR, "\x00"},
		tk{token.IDENTIFIER, "b"},
	)
}

// ---------------------------------------------------------------------------
// Punctuation and operators
// ---------------------------------------------------------------------------

func TestSingleCharTokens(t *testing.T) {
	cases := []tk{
		{token.LEFT_PAREN, "("},
		{token.RIGHT_PAREN, ")"},
		{token.LEFT_BRACE, "{"},
		{token.RIGHT_BRACE, "}"},
		{token.COMMA, ","},
		{token.DOT, "."},
		{token.MINUS, "-"},
		{token.PLUS, "+"},
		{token.SEMICOLON, ";"},
		{token.SLASH, "/"},
		{token.STAR, "*"},
	}
	for _, c := range cases {
		t.Run(c.lexeme, func(t *testing.T) {
			expectTokens(t, c.lexeme, c)
		})
	}
}

func TestOneOrTwoCharOperators(t *testing.T) {
	cases := []tk{
		{token.BANG, "!"},
		{token.BANG_EQUAL, "!="},
		{token.EQUAL, "="},
		{token.EQUAL_EQUAL, "=="},
		{token.LESS, "<"},
		{token.LESS_EQUAL, "<="},
		{token.GREATER, ">"},
		{token.GREATER_EQUAL, ">="},
	}
	for _, c := range cases {
		t.Run(c.lexeme, func(t *testing.T) {
			expectTokens(t, c.lexeme, c)
		})
	}
}

// "===" is "==" then "=": match() takes at most one extra byte.
func TestOperatorGreedyOnce(t *testing.T) {
	expectTokens(t, "===",
		tk{token.EQUAL_EQUAL, "=="},
		tk{token.EQUAL, "="},
	)
}

// ---------------------------------------------------------------------------
// Identifiers and keywords
// ---------------------------------------------------------------------------

func TestIdentifiers(t *testing.T) {
	for _, src := range []string{
		"x", "abc", "_foo", "foo_", "foo_bar", "foo123", "___",
	} {
		t.Run(src, func(t *testing.T) {
			expectTokens(t, src, tk{token.IDENTIFIER, src})
		})
	}
}

func TestKeywords(t *testing.T) {
	cases := []tk{
		{token.AND, "and"},
		{token.CLASS, "class"},
		{token.ELSE, "else"},
		{token.FALSE, "false"},
		{token.FOR, "for"},
		{token.FUN, "fun"},
		{token.IF, "if"},
		{token.NIL, "nil"},
		{token.OR, "or"},
		{token.PRINT, "print"},
		{token.RETURN, "return"},
		{token.SUPER, "super"},
		{token.THIS, "this"},
		{token.TRUE, "true"},
		{token.VAR, "var"},
		{token.WHILE, "while"},
	}
	for _, c := range cases {
		t.Run(c.lexeme, func(t *testing.T) {
			expectTokens(t, c.lexeme, c)
		})
	}
}

// Words that look like keywords but are not: longer, shorter, a single
// letter, or wrong case. All must be IDENTIFIER.
func TestKeywordLookalikes(t *testing.T) {
	cases := []string{
		// longer than a keyword
		"andor", "classroom", "elsewhere", "falsely", "forward",
		"funny", "iffy", "nile", "orange", "printer", "returning",
		"superb", "thistle", "truest", "vary", "whiley", "class1", "var_",
		// shorter than a keyword
		"an", "cla", "el", "fal", "fo", "fu", "ni", "pri", "ret",
		"sup", "th", "tr", "va", "whil",
		// single letters
		"a", "c", "e", "f", "i", "n", "o", "p", "r", "s", "t", "v", "w",
		// keywords are lowercase only
		"AND", "And", "TRUE", "Nil",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectTokens(t, src, tk{token.IDENTIFIER, src})
		})
	}
}

// A keyword in the middle of the source, not at offset 0. Catches keyword
// checks that slice from 0 instead of from the token's start.
func TestKeywordNotAtStart(t *testing.T) {
	expectTokens(t, "x and y",
		tk{token.IDENTIFIER, "x"},
		tk{token.AND, "and"},
		tk{token.IDENTIFIER, "y"},
	)
}

// ---------------------------------------------------------------------------
// Numbers
// ---------------------------------------------------------------------------

func TestNumbers(t *testing.T) {
	for _, src := range []string{"0", "123", "1.5", "123.456"} {
		t.Run(src, func(t *testing.T) {
			expectTokens(t, src, tk{token.NUMBER, src})
		})
	}
}

// The '.' is only part of the number when a digit follows it.
func TestNumberTrailingDot(t *testing.T) {
	expectTokens(t, "123.",
		tk{token.NUMBER, "123"},
		tk{token.DOT, "."},
	)
}

// A number has at most one fractional part.
func TestNumberMultipleDots(t *testing.T) {
	expectTokens(t, "1.2.3",
		tk{token.NUMBER, "1.2"},
		tk{token.DOT, "."},
		tk{token.NUMBER, "3"},
	)
}

// ---------------------------------------------------------------------------
// Strings
// ---------------------------------------------------------------------------

func TestStrings(t *testing.T) {
	for _, src := range []string{`"abc"`, `""`, "\"line1\nline2\""} {
		t.Run(src, func(t *testing.T) {
			expectTokens(t, src, tk{token.STRING, src})
		})
	}
}

// An escaped quote does not end the string. The scanner only skips over
// escapes; decoding them happens later, when building the AST.
func TestStringEscapedQuote(t *testing.T) {
	expectTokens(t, `"a\"b"`, tk{token.STRING, `"a\"b"`})
}

// An unterminated string is one ERROR token spanning from the opening
// quote to the end of the source, plus one diagnostic with the same span.
func TestStringUnterminated(t *testing.T) {
	for _, src := range []string{`"abc`, "\"abc\ndef"} {
		t.Run(src, func(t *testing.T) {
			r := expectTokens(t, src, tk{token.ERROR, src})
			expectOneDiagnostic(t, r, 0, uint32(len(src)))
		})
	}
}

// ---------------------------------------------------------------------------
// Whitespace and comments
// ---------------------------------------------------------------------------

func TestWhitespaceBetweenTokens(t *testing.T) {
	for _, src := range []string{"+  +", "+\t\t+", "+\r\r+", "+ \n +"} {
		t.Run(fmt.Sprintf("%q", src), func(t *testing.T) {
			expectTokens(t, src, tk{token.PLUS, "+"}, tk{token.PLUS, "+"})
		})
	}
}

func TestComments(t *testing.T) {
	expectTokens(t, "// a comment\n+", tk{token.PLUS, "+"})
	expectTokens(t, "+ // trailing comment", tk{token.PLUS, "+"})
	expectTokens(t, "// comment with no newline at the end")
}

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

// expectOneDiagnostic checks there is exactly one diagnostic, with the
// given span and a non-empty message.
func expectOneDiagnostic(t *testing.T, r result, start, end uint32) {
	t.Helper()
	if len(r.diags) != 1 {
		t.Fatalf("got %d diagnostics, want 1: %+v", len(r.diags), r.diags)
	}
	d := r.diags[0]
	if d.Start != start || d.End != end {
		t.Errorf("diagnostic span = [%d,%d), want [%d,%d)", d.Start, d.End, start, end)
	}
	if strings.TrimSpace(d.Message) == "" {
		t.Error("diagnostic has an empty message")
	}
}

// An unexpected byte becomes an ERROR token covering just that byte, and
// scanning continues afterwards.
func TestUnexpectedByte(t *testing.T) {
	r := expectTokens(t, "(+@+)",
		tk{token.LEFT_PAREN, "("},
		tk{token.PLUS, "+"},
		tk{token.ERROR, "@"},
		tk{token.PLUS, "+"},
		tk{token.RIGHT_PAREN, ")"},
	)
	expectOneDiagnostic(t, r, 2, 3)
}

func TestSeveralErrorsAllReported(t *testing.T) {
	r := expectTokens(t, "@ # $",
		tk{token.ERROR, "@"},
		tk{token.ERROR, "#"},
		tk{token.ERROR, "$"},
	)
	if len(r.diags) != 3 {
		t.Errorf("got %d diagnostics, want 3", len(r.diags))
	}
}

func TestValidSourceHasNoDiagnostics(t *testing.T) {
	r := scan(t, "var x = (1 + 2.5) * y;")
	if len(r.diags) != 0 {
		t.Errorf("got diagnostics for valid source: %+v", r.diags)
	}
}

// ---------------------------------------------------------------------------
// Line tracking (lineStarts)
// ---------------------------------------------------------------------------

func TestLineStarts(t *testing.T) {
	cases := []struct {
		src  string
		want []uint32
	}{
		{"", []uint32{0}},
		{"abc", []uint32{0}},
		{"a\nb\n", []uint32{0, 2, 4}},
		{"\n\n", []uint32{0, 1, 2}},
		{"// comment\nx", []uint32{0, 11}},    // newline ending a comment
		{"\"a\nb\"\nx", []uint32{0, 3, 6}},    // newline inside a string
		{"f\"a\n{x}\"\ny", []uint32{0, 4, 9}}, // newline inside f-string text
		{"\"\n\n\"", []uint32{0, 2, 3}},       // two newlines in a row, string
		{"f\"\n\n\"", []uint32{0, 3, 4}},      // two newlines in a row, f-string
		{"\"a\\\nb\"", []uint32{0, 4}},        // escaped newline in a string
		{"f\"a\\\nb\"", []uint32{0, 5}},       // escaped newline in an f-string
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%q", c.src), func(t *testing.T) {
			r := scan(t, c.src)
			if fmt.Sprint(r.lineStarts) != fmt.Sprint(c.want) {
				t.Errorf("lineStarts = %v, want %v", r.lineStarts, c.want)
			}
		})
	}
}

func TestTokenLines(t *testing.T) {
	r := expectTokens(t, "+\n+\n\n@",
		tk{token.PLUS, "+"},
		tk{token.PLUS, "+"},
		tk{token.ERROR, "@"},
	)
	for i, want := range []int{1, 2, 4} {
		if got := r.lineOf(i); got != want {
			t.Errorf("token %d is on line %d, want %d", i, got, want)
		}
	}
}

// A token's position is where it STARTS. A string spanning lines 1-2 is on
// line 1, and the token after it is on line 3.
func TestMultilineStringPosition(t *testing.T) {
	r := expectTokens(t, "\"line1\nline2\"\n+",
		tk{token.STRING, "\"line1\nline2\""},
		tk{token.PLUS, "+"},
	)
	if got := r.lineOf(0); got != 1 {
		t.Errorf("string is on line %d, want 1 (the line it starts on)", got)
	}
	if got := r.lineOf(1); got != 3 {
		t.Errorf("'+' is on line %d, want 3", got)
	}
}

// ---------------------------------------------------------------------------
// f-strings
//
// Token scheme: text pieces carry the braces/quotes at their ends.
//   f"a {x} b {y} c"  →  F_STRING_START `f"a {`  x  F_STRING_MID `} b {`
//                        y  F_STRING_END `} c"`
// ---------------------------------------------------------------------------

// An f-string with no holes is a single F_STRING_END token.
func TestFStringNoInterpolation(t *testing.T) {
	expectTokens(t, `f"hello"`, tk{token.F_STRING_END, `f"hello"`})
}

func TestFStringSingleInterpolation(t *testing.T) {
	expectTokens(t, `f"a {b} c"`,
		tk{token.F_STRING_START, `f"a {`},
		tk{token.IDENTIFIER, `b`},
		tk{token.F_STRING_END, `} c"`},
	)
}

func TestFStringMultipleInterpolations(t *testing.T) {
	expectTokens(t, `f"this {foo} text {bar} end"`,
		tk{token.F_STRING_START, `f"this {`},
		tk{token.IDENTIFIER, `foo`},
		tk{token.F_STRING_MID, `} text {`},
		tk{token.IDENTIFIER, `bar`},
		tk{token.F_STRING_END, `} end"`},
	)
}

func TestFStringEmptyInterpolation(t *testing.T) {
	expectTokens(t, `f"{}"`,
		tk{token.F_STRING_START, `f"{`},
		tk{token.F_STRING_END, `}"`},
	)
}

// Only `f` immediately followed by `"` starts an f-string.
func TestFStringNeedsAdjacentQuote(t *testing.T) {
	expectTokens(t, `f + "str"`,
		tk{token.IDENTIFIER, "f"},
		tk{token.PLUS, "+"},
		tk{token.STRING, `"str"`},
	)
}

// Braces inside a hole are ordinary braces. Only the '}' that balances the
// hole's '{' closes it. (A single interpMode bool gets this wrong.)
func TestFStringBracesInsideHole(t *testing.T) {
	expectTokens(t, `f"a{ {1} }b"`,
		tk{token.F_STRING_START, `f"a{`},
		tk{token.LEFT_BRACE, "{"},
		tk{token.NUMBER, "1"},
		tk{token.RIGHT_BRACE, "}"},
		tk{token.F_STRING_END, `}b"`},
	)
}

// An f-string inside a hole of another f-string.
func TestFStringNested(t *testing.T) {
	expectTokens(t, `f"<{ f"[{x}]" }>"`,
		tk{token.F_STRING_START, `f"<{`},
		tk{token.F_STRING_START, `f"[{`},
		tk{token.IDENTIFIER, "x"},
		tk{token.F_STRING_END, `}]"`},
		tk{token.F_STRING_END, `}>"`},
	)
}

// A '}' with no open f-string is a plain RIGHT_BRACE.
func TestRightBraceOutsideFString(t *testing.T) {
	expectTokens(t, "}", tk{token.RIGHT_BRACE, "}"})
}

// No closing quote and no hole: one ERROR spanning to the end.
func TestFStringUnterminated(t *testing.T) {
	r := expectTokens(t, `f"abc`, tk{token.ERROR, `f"abc`})
	expectOneDiagnostic(t, r, 0, 5)
}

// Source ends inside a hole: the tokens so far, then exactly one
// diagnostic. Where the ERROR token sits is up to the scanner, so this
// only checks the kinds and the diagnostic count.
func TestFStringUnterminatedInsideHole(t *testing.T) {
	r := scan(t, `f"abc {expr`)
	var kinds []token.Kind
	for _, tok := range r.toks {
		kinds = append(kinds, tok.Type)
	}
	want := []token.Kind{token.F_STRING_START, token.IDENTIFIER, token.ERROR, token.EOF}
	if fmt.Sprint(kinds) != fmt.Sprint(want) {
		t.Errorf("kinds = %v, want %v", kinds, want)
	}
	if len(r.diags) != 1 {
		t.Errorf("got %d diagnostics, want 1", len(r.diags))
	}
}

// After an unterminated f-string the scanner state is reset: a later '}'
// on a fresh scan is a plain RIGHT_BRACE. (Separate scans share nothing.)
func TestScansAreIndependent(t *testing.T) {
	scan(t, `f"abc {expr`)
	expectTokens(t, "}", tk{token.RIGHT_BRACE, "}"})
}

// ---------------------------------------------------------------------------
// Newlines inside strings
// ---------------------------------------------------------------------------

// The byte right after a newline must still be checked: here it is the
// closing quote.
func TestStringClosesRightAfterNewline(t *testing.T) {
	expectTokens(t, "\"a\n\"", tk{token.STRING, "\"a\n\""})
}

// Source ends right after a newline inside a string: an error, not a panic.
func TestUnterminatedEndingInNewline(t *testing.T) {
	r := expectTokens(t, "\"a\n", tk{token.ERROR, "\"a\n"})
	expectOneDiagnostic(t, r, 0, 3)

	r = expectTokens(t, "f\"a\n", tk{token.ERROR, "f\"a\n"})
	expectOneDiagnostic(t, r, 0, 4)
}

// ---------------------------------------------------------------------------
// Escapes: a backslash skips the next byte, in strings and f-strings.
// The scanner only skips them; decoding happens when building the AST.
// ---------------------------------------------------------------------------

// `"a\\"` is an escaped backslash followed by the closing quote.
func TestStringEscapedBackslash(t *testing.T) {
	expectTokens(t, `"a\\"`, tk{token.STRING, `"a\\"`})
}

// A backslash as the very last byte: unterminated, and must not panic.
func TestStringBackslashAtEnd(t *testing.T) {
	r := expectTokens(t, `"abc\`, tk{token.ERROR, `"abc\`})
	expectOneDiagnostic(t, r, 0, 5)
}

// \{ and \} are literal braces in f-string text: no hole is opened.
func TestFStringEscapedBraces(t *testing.T) {
	expectTokens(t, `f"\{x\}"`, tk{token.F_STRING_END, `f"\{x\}"`})
}

func TestFStringEscapedQuote(t *testing.T) {
	expectTokens(t, `f"a\"b"`, tk{token.F_STRING_END, `f"a\"b"`})
}

// ---------------------------------------------------------------------------
// f-string stack hygiene
// ---------------------------------------------------------------------------

// Closing a hole must pop the stack. After the f-string ends, a '}' is a
// plain RIGHT_BRACE again.
func TestFStringStackPopsOnClose(t *testing.T) {
	expectTokens(t, `f"{x}" }`,
		tk{token.F_STRING_START, `f"{`},
		tk{token.IDENTIFIER, "x"},
		tk{token.F_STRING_END, `}"`},
		tk{token.RIGHT_BRACE, "}"},
	)
}

// Complete f-strings must leave nothing open at EOF: no diagnostics.
func TestFStringValidHasNoDiagnostics(t *testing.T) {
	for _, src := range []string{
		`f"{x}"`,
		`f"a{b}c{d}e"`,
		`f"a{ {1} }b"`,
		`f"<{ f"[{x}]" }>"`,
	} {
		t.Run(src, func(t *testing.T) {
			r := scan(t, src)
			if len(r.diags) != 0 {
				t.Errorf("got diagnostics for a valid f-string: %+v", r.diags)
			}
		})
	}
}

// A quoted string inside a hole is one STRING token; its '}' never reaches
// the brace logic.
func TestFStringStringInsideHole(t *testing.T) {
	expectTokens(t, `f"{ "}" }"`,
		tk{token.F_STRING_START, `f"{`},
		tk{token.STRING, `"}"`},
		tk{token.F_STRING_END, `}"`},
	)
}
