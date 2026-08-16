---
date: 2026-08-16
phase: lexer
files: [internals/lexer/scanner.go, internals/lexer/token.go, internals/compiler/compiler.go, internals/compiler/chunks.go, internals/debug/debug.go, internals/vm/vm.go, internals/repl/repl.go, internals/value/value.go, cmd/glox/main.go, go.mod, .claude/CLAUDE.md, .claude/output-styles/mentor.md, .claude/settings.json]
commit: fddfc19
---

# Scanner: keywords/identifiers/numbers/strings/two-char ops (WIP, does not build) + module path typo fix + agent config updates

## What
This working tree mixes three unrelated concerns; recorded together per
user direction, split candidates noted below.

**Scanner feature work (`internals/lexer/scanner.go`, `token.go`,
`internals/compiler/compiler.go`) — incomplete, build fails:**
- `ScanToken` now calls `skipWhiteSpace()` first, dispatches to
  `makeIdentifier()`/`makeNumber()` via `isAlpha`/`isDigit`, and adds
  `!`, `=`, `<`, `>` two-char lookahead (`match`) plus `"` → `string()`.
- `makeToken` length fixed to `current − start` (was already fixed in a
  prior session per the stale `2026-08-01` entry; re-verified here).
- `identifierType()`/`checkKeyword()` added for the keyword table (`and`,
  `class`, `else`, `if`, `nil`, `or`, `print`, `return`, `super`, `var`,
  `while`) but are broken: `identifierType()` is defined as a method
  `(s *Scanner) identifierType()` yet called as a free function at
  `scanner.go:168`; `checkKeyword` (`scanner.go:222-227`) has an
  incomplete condition — `... || (1) { }` (untyped int `1` where a bool
  is required, translated from clox's C truthy idiom), an empty `if`
  body, and a missing `return` on the fallthrough path.
- `token.go`: `Token` fields exported (`Type/Start/Length/Line`); added
  `GetLexme()` reconstructing the lexeme via `unsafe.String`.
- `compiler.go`: `Compile` now loops `ScanToken()` until `EOF`, printing
  `Scanner Error: ...` on `lexer.ERROR` tokens instead of panicking.

**Module path typo fix (mechanical, unrelated to the above):**
- `go-bytecode-interpter` → `go-bytecode-interpreter` in `go.mod`,
  `cmd/glox/main.go`, and every internal import in `chunks.go`,
  `debug.go`, `repl.go`, `value.go`, `vm.go`.

**Agent/tooling config (unrelated to code):**
- `.claude/CLAUDE.md`: condensed the project-contract section; added the
  `/issue` ↔ `ISSUES.md` cross-reference and the ISSUE-NNN citation rule.
- `.claude/output-styles/mentor.md`: reorganized (209 → 116 net lines).
- `.claude/settings.json`: added a `deny` list ahead of `allows`,
  switched `outputStyle` to `compiler-mentor`, added `enabledPlugins`.

## Why
Continuing scanner work from the `2026-08-01` session (export fields,
fix length calc) toward a full single-character + keyword scanner. The
module path typo (`interpter`) was carried from the initial scaffold
commit and is unrelated — fixed opportunistically while touching these
files (closes the file-list half of ISSUE-012; `vm_bench_test.go` still
has the stale import and is untouched by this diff). Config changes
reflect agent-workflow adjustments made this session, also unrelated to
the scanner work.

This diff also fixes ISSUE-010 (`Line` unset), ISSUE-011 (`Compile`'s
infinite loop), ISSUE-013/014/015 (`skipWhiteSpace` hangs/single-byte/
comment-newline bugs) — all closed in `.claude/ISSUES.md` this session —
but introduces a new defect, ISSUE-016 (`checkKeyword`/`identifierType`),
which is why the build still fails.

## How
Keyword dispatch follows the clox trie-by-first-letter approach
(`identifierType` switches on `s.start`'s first byte, delegates length/
suffix check to `checkKeyword`) rather than a map — but the Go port of
the C `||`-as-truthy check was left unfinished mid-translation.

## Evidence
- `go build ./...` — **fails**:
  ```
  internals\lexer\scanner.go:168:21: undefined: identifierType
  internals\lexer\scanner.go:223:5: invalid operation: (calcPtrDiff(s.start, s.current) == (start + length)) || (1) (mismatched types untyped bool and untyped int)
  internals\lexer\scanner.go:227:1: missing return
  ```
- `go vet ./...` — fails on the same build error, plus a pre-existing
  stale import in `internals/vm/vm_bench_test.go` (still references the
  old `go-bytecode-interpter` module path — not touched by this diff,
  will break once the rename lands).
- Not committable as-is: `checkKeyword` needs the C `||` idiom translated
  correctly to a Go bool expression, and `identifierType()` needs a
  receiver at its call site.

## Stuck points
`checkKeyword`'s condition was left as a direct transliteration of
clox's `memcmp`-based check: `length == start + length && memcmp(...)`
in C treats any nonzero `memcmp` result as truthy-continue; the literal
`|| (1)` is a leftover fragment from that translation and needs to
become a real boolean (e.g. compare `memCompare(...)` — already defined
below but unused — against 0).
