---
date: 2026-08-20
phase: lexer
files: [internals/lexer/scanner.go, internals/lexer/token.go, internals/lexer/tokentype_string.go, internals/lexer/scanner_test.go, internals/compiler/compiler.go, examples/scanner_text.glox, Makefile, .claude/ISSUES.md]
commit: 12540a9
---

# Add f-string interpolation scanning

## What
- `internals/lexer/token.go:59-62` — three new `TokenType` values,
  `F_STRING_START`, `F_STRING_MID`, `F_STRING_END`, inserted between
  `WHILE` and `ERROR`; `tokentype_string.go` regenerated to match (stringer
  index table widened `uint8` → `uint16`).
- `internals/lexer/scanner.go`:
  - `Scanner.interpMode bool` field (line 11), `false` by default.
  - `ScanToken()` (lines 37-40): `c == 'f' && s.peek() == '"'` dispatches to
    `fstringStart()` before the `isAlpha` check, so `f"..."` never scans as
    an identifier.
  - New `fstringStart()` (167-178) and `fstringScan()` (180-188): a
    plain f-string with no `{` short-circuits straight to a single
    `F_STRING_END` token; one with `{` emits `F_STRING_START` and flips
    `interpMode`.
  - `case '}':` (55-69) gated on `interpMode`: closes the interpolation as
    `F_STRING_END` if the next literal chunk ends at `"`, or opens the next
    one as `F_STRING_MID` if it ends at another `{`; falls through to plain
    `RIGHT_BRACE` when `interpMode` is false.
  - Rider fix: `skipWhiteSpace()`'s `case '/':` (136-146) moved `return`
    outside the comment `if/else`.
- `internals/compiler/compiler.go:29-32` — rider fix: scan-error path now
  uses `token.GetLexme()` instead of `%v` on the raw token, and `return`s
  instead of falling through to `fmt.Println(token)`.
- `Makefile:13` — `run:` target passes `$(ARGS)` through (dev convenience).
- `examples/scanner_text.glox` — fixture replaced with an f-string sample
  (`f"this {is} a text {and} this {another} text" + "this is a string"`).
- `internals/lexer/scanner_test.go` (+188 lines, new, written by the
  assistant on request) — 8 new tests: no-interpolation, single, multiple,
  and empty (`{}`) interpolations; multiline line-tracking across both
  `fstringScan` call sites; an adjacency pin (`f + "str"` must not misfire);
  and two exploratory edge-case tests for unterminated f-strings.
- `.claude/ISSUES.md` — new `ISSUE-020` (open), filed after the exploratory
  tests surfaced a real regression (see Stuck points).

## Why
Next scanner feature on the roadmap: lexical support for f-string
interpolation, so the parser/compiler eventually has tokens to build string
concatenation from. The two rider fixes (Makefile, compiler.go error
reporting) were made in the same session while touching this code and are
bundled into this one commit/entry by decision, rather than split out.

## How
Three-token scheme, no dedicated token for the embedded expression itself —
between `F_STRING_START`/`F_STRING_MID` and the next `F_STRING_MID`/
`F_STRING_END`, the scanner is in ordinary mode and the embedded expression
is tokenized by the normal `ScanToken()` path (identifiers, numbers,
operators, etc.), which is what lets `f"a {b + 1} c"` scan `b`, `PLUS`, `1`
as regular tokens with zero extra scanner state beyond the one `interpMode`
bool. `case '}':` only branches on this new logic when `interpMode` is
true, so ordinary brace-matching (`RIGHT_BRACE`) is untouched when there's
no f-string in progress. A plain f-string with no `{` deliberately skips
`F_STRING_START` entirely and emits a single `F_STRING_END` — the parser
will need to treat "just `F_STRING_END`" and "`F_STRING_START` ... 
`F_STRING_END`" as the two valid shapes.

Test-writing (assistant, on request) followed the existing table-driven /
scenario conventions in `scanner_test.go`, but added one new helper —
`assertFStringSequence`, asserting via `Token.GetLexme()` rather than
manual `Length`/`byteAt` arithmetic — because f-string lexeme boundaries are
irregular (each `F_STRING_MID`/`END` includes the `}`/`"` that *closed the
previous* segment), making manual byte counting error-prone. Traced every
test case against the actual code by hand before running, to avoid encoding
wrong assumptions as passing tests; all 8 passed on the first run, which
confirms the traces (documented inline in the test file) match real
scanner behavior.

## Evidence
```
$ go build ./...
(exit 0)

$ go vet ./internals/lexer/... ./internals/compiler/...
(exit 0 — full `go vet ./...` fails only on pre-existing, unrelated
ISSUE-019 and ISSUE-006)

$ go test ./internals/lexer/... -run TestScanToken_FString -v
--- PASS: TestScanToken_FString_NoInterpolation (0.00s)
--- PASS: TestScanToken_FString_SingleInterpolation (0.00s)
--- PASS: TestScanToken_FString_MultipleInterpolations (0.00s)
--- PASS: TestScanToken_FString_EmptyInterpolation (0.00s)
--- PASS: TestScanToken_FString_Multiline (0.00s)
--- PASS: TestScanToken_FString_AdjacentFIdentifier (0.00s)
--- PASS: TestScanToken_FString_UnterminatedMidInterpolation (0.00s)
--- PASS: TestScanToken_FString_UnterminatedNoClose (0.00s)
PASS

$ go test ./internals/lexer/... -v
... (all prior tests PASS) ...
--- FAIL: TestScanToken_SkipComment (0.00s)
    scanner_test.go:474: Type = ERROR, want PLUS (comment not fully skipped)
... (all 8 new FString tests PASS) ...
FAIL

$ go test ./internals/compiler/...
ok
```

## Stuck points
Running the full `internals/lexer` suite (not just the new tests) surfaced
that the `skipWhiteSpace()` rider fix broke comment-skipping — a
pre-existing, previously-passing test (`TestScanToken_SkipComment`) now
fails with an `ERROR` token instead of `PLUS`. Traced and filed as
**ISSUE-020** (open) rather than fixed here, since it's implementation work.
This is exactly why the plan called for running the *whole* package suite,
not just the new f-string tests, before writing this entry.
