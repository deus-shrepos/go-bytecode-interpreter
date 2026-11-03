---
date: 2026-11-03
phase: lexer
files: [internals/token/token.go, internals/token/token_test.go, internals/lexer/scanner.go, internals/lexer/scanner_test.go]
commit: pending
---

# Rewrite tokens as offset spans and scanner as a whole-file pass

## What
- `token.go`: `Token` is now `{Start, End uint32; Type Kind}` (12 B), a span
  into the source; dropped `unsafe.Pointer`/`Length`/`Line` and `Print`.
  Added `Diagnostics{Start, End, Message}`, `Lexeme(source)` (returns `""`
  for zero-length spans, else `unsafe.String` aliasing `source`), `Len()`,
  and `FindLineColumn(lines, offset)` — binary search over line starts,
  1-based line, 0-based byte column.
- `scanner.go`: cursor is `start`/`current uint32` offsets; `isAtEnd` is
  `current >= len(source)`; `peek`/`peekNext` return 0 past the end.
- `ScanTokens(source) (tokens, lineStarts, diags)` (`:29`): loop
  scan → emit → break on EOF; `scanToken` returns a `token.Kind` and
  `EmitToken` (`:136`) is the single token append site.
- Newlines recorded in exactly one place: `advance()` (`:350`) appends the
  offset after each `\n` to `lines` (pre-sized with `bytes.Count`).
- `ReportError` appends a diagnostic and returns `ERROR`; scanning continues.
- f-strings: `interpMode bool` → `fstack []uint32` (`:12`), one entry per open
  hole holding its count of open code braces; push in `fstringText` (`:213`),
  `++`/`--`/pop in the `{`/`}` cases (`:73-85`), clear + error at EOF inside
  a hole (`:48`). `fstringText(isFirst)` replaces `fstringStart`.
- Escapes: `string()` and `fstringText()` skip `\` + next byte (`\"`, `\\`,
  `\{`, `\}`), guarded at end of input.
- `checkKeyword` slices `source[start+k:current]`; 'f'/'t' second-byte reads
  guarded by token length.
- Tests: `token_test.go` (size, `Lexeme`, `FindLineColumn` table) and a
  rewritten `scanner_test.go` (whole-stream `(kind, lexeme)` checks, a
  any-input invariant suite, line starts, errors, escapes, f-string nesting).

## Why
Moving from clox single-pass compile to `Source → []Token → flat AST →
lowering`. `ast.Node` refers to tokens by index, so tokens now live for the
whole compile and their size/ownership matters. Review also found two
committed defects, both fixed here: ISSUE-028 (NUL-sentinel `isAtEnd` read
past the slice) and ISSUE-029 (`interpMode` bool: silent unterminated
f-strings, broken nesting).

## How
Line/col derived from `Start` on the (cold) error path instead of stored per
token: 12 B vs 20 B per token, one source of truth, and multi-line tokens get
their start position for free. Tokens, line starts and diagnostics stay plain
Go slices, not the `memory.Space` arena: each is one contiguous pointer-free
(noscan) block, and `Diagnostics.Message` holds a Go pointer that must not
live in GC-invisible memory. Chose "scanToken returns a Kind, one emit site"
over per-branch `emit`+`return` so double tokens and double EOF are
structurally impossible. Literal braces in f-string text are `\{`/`\}` (one
backslash rule for strings and f-strings). Human wrote all non-test code;
assistant reviewed, traced bugs, and wrote both test files.

## Evidence
```
$ go vet ./internals/lexer ./internals/token
(clean)
$ go test ./internals/lexer ./internals/token -count=1
ok  	go-bytecode-interpreter/internals/lexer	0.639s
ok  	go-bytecode-interpreter/internals/token	0.627s
(197 tests/subtests passing)
$ go build ./...
internals\parser\parser.go:1:1: expected 'package', found 'EOF'
```
`go build ./...` is broken by the uncommitted parser/compiler WIP, which
still targets the old token API — not part of this commit. No benchmark run;
the 48 B → 12 B token-size benefit is unmeasured.

## Stuck points
- Converting `return makeToken(k)` to `emit(k)` dropped the returns that made
  branches exclusive → fall-through into later branches. Fixed by option B.
- `newLine()` before `advance()`, then a double advance after `\n`, then an
  escaped `\n` skipping the record — three bugs of one shape; resolved by
  moving recording into `advance()`.
- `'}'` decremented instead of popping; `uint32` wrapped to 4294967295,
  leaking a hole so valid f-strings reported "unterminated".
- Known leftovers: stringer directive says `-type=Type` (type is `Kind`);
  commented-out code at `scanner.go:154,208`; typo "f-stirng" (`:228`);
  `NewToken`/`Kind()` still present.
