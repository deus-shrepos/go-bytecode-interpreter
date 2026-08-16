---
date: 2026-08-01
phase: lexer
files: [internals/lexer/token.go, internals/lexer/scanner.go, internals/compiler/compiler.go, examples/scanner_text.glox]
commit: pending
---

# Export Token fields, fix lexeme length calc, wire scanner errors into compiler

## What
- `internals/lexer/token.go`: exported all `Token` fields
  (`ttype/start/length/line` → `Type/Start/Length/Line`); added
  `getLexeme()` reconstructing the lexeme string from `Start`/`Length`
  via `unsafe.Slice`.
- `internals/lexer/scanner.go:60`: rewrote `makeToken`'s length as
  `current − start` pointer difference; previous version dereferenced
  `unsafe.Add(s.start, -uintptr(s.current))` as an `*int`, reading
  arbitrary memory instead of computing a length. `errorToken` updated
  to the exported field names.
- `internals/compiler/compiler.go:15`: `Compile` now captures the token
  from `ScanToken()` and panics on `lexer.ERROR` — first cross-package
  consumer of `Token`, and the reason the fields had to be exported.
- `examples/scanner_text.glox` (new): tiny scanner input `(+++++++)`
  for manual testing.

## Why
One work session pushing the lexer → compiler handoff forward: the
compiler package needed to read `Token.Type`, which forced the field
exports; the broken length computation in `makeToken` and the error-token
propagation were fixed/added in the same pass toward that milestone.

## How
Length is now the pointer difference `current − start` converted through
`uintptr` — the standard clox `(int)(scanner.current - scanner.start)`
translated to Go's `unsafe.Add`. Fields were exported rather than adding
accessor methods, matching the C struct's open layout. Error handling in
`Compile` is a placeholder panic until a real error path exists.

## Evidence
- `go build ./internals/lexer/ ./internals/compiler/` — passes.
- `go test` on both packages: `[no test files]`.
- `go vet ./...` fails only on pre-existing issues outside this diff
  (`internals/repl/repl.go:18` unused `fmt.Errorf`;
  `internals/vm/vm_bench_test.go:22` stale `Interpret` call).
