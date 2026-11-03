package token

import (
	"testing"
	"unsafe"
)

// Tokens live in one big slice for the whole compile, so their size matters.
// Start(4) + End(4) + Kind(1) + padding(3) = 12 bytes.
func TestTokenSize(t *testing.T) {
	if got := unsafe.Sizeof(Token{}); got != 12 {
		t.Errorf("Token size = %d bytes, want 12", got)
	}
}

func TestLexeme(t *testing.T) {
	src := []byte("let x = 42")

	tests := []struct {
		name  string
		start uint32
		end   uint32
		want  string
	}{
		{"first word", 0, 3, "let"},
		{"single byte", 4, 5, "x"},
		{"last word", 8, 10, "42"},
		{"whole source", 0, 10, "let x = 42"},
		{"empty in the middle", 4, 4, ""},
		{"EOF: empty at len(src)", 10, 10, ""}, // must not panic
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok := Token{Start: tt.start, End: tt.end}
			if got := tok.Lexeme(src); got != tt.want {
				t.Errorf("Lexeme() = %q, want %q", got, tt.want)
			}
			if got := tok.Len(); got != len(tt.want) {
				t.Errorf("Len() = %d, want %d", got, len(tt.want))
			}
		})
	}
}

// Source used for line/column tests:
//
//	offset:  0  1  2  3  4  5  6  7  8
//	byte:    a  b  \n c  d  \n \n e  f
//
// Line 1 = "ab", line 2 = "cd", line 3 = "" (empty), line 4 = "ef".
// Lines are 1-based, columns are 0-based and counted in bytes.
func TestFindLineColumn(t *testing.T) {
	lineStarts := []uint32{0, 3, 6, 7}

	tests := []struct {
		name     string
		offset   uint32
		wantLine int
		wantCol  int
	}{
		{"first byte of file", 0, 1, 0},
		{"newline belongs to the line it ends", 2, 1, 2},
		{"middle of a line", 4, 2, 1},
		{"empty line (offset equals a line start)", 6, 3, 0},
		{"first byte of last line", 7, 4, 0},
		{"EOF (offset == len(src))", 9, 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			line, col := FindLineColumn(lineStarts, tt.offset)
			if line != tt.wantLine || col != tt.wantCol {
				t.Errorf("FindLineColumn(%d) = (%d, %d), want (%d, %d)",
					tt.offset, line, col, tt.wantLine, tt.wantCol)
			}
		})
	}
}

// A source with no newlines has exactly one line.
func TestFindLineColumn_SingleLine(t *testing.T) {
	line, col := FindLineColumn([]uint32{0}, 5)
	if line != 1 || col != 5 {
		t.Errorf("FindLineColumn(5) = (%d, %d), want (1, 5)", line, col)
	}
}
