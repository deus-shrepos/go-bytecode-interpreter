---
date: 2026-08-29
phase: compiler
files: [internals/compiler/compiler.go, internals/compiler/precedence.go,
  internals/compiler/chunks.go, internals/vm/vm.go, internals/vm/operations.go,
  internals/repl/repl.go, internals/lexer/scanner_test.go,
  examples/scanner_text.glox, .claude/ISSUES.md, pratt-parser.md, .gitignore]
commit: pending
---

# WIP: Pratt-parser scaffolding for expression compilation (build broken)

## What
- `internals/compiler/precedence.go` (new) — `Precedence` enum
  (`PrecNone`..`PrecPrimary`), `ParseRule`/`ParseFunc` types, and a `rules`
  table keyed by `lexer.TokenType`.
- `compiler.go` — adds `emitByteCode`/`emitBytes`/`emitReturn`/`endCompiler`,
  `grouping`/`unary`/`Binary`/`number`/`makeConstant`, and a `parsePrecedence`
  stub; `Compile()` now calls `endCompiler()` after the expression;
  `Compiler` gains a `compilingChunk` field.
- `chunks.go` — `Chunk.Consts` changed from `[]value.Value` to `[]float64`;
  `addConst` renamed to exported `AddConstant`; `WriteConstant` now returns
  the constant's index (`int`) instead of nothing.
- `vm.go` / `operations.go` — `Value` replaced with `float64` throughout
  (`VM.Stack`, `Push`/`Pop`, `popTwoNumbers`); `Interpret` updated to the
  3-arg `NewCompiler(arena, chunk, source)` / no-arg `Compile()` shape.
- `repl.go` — `LoadProgramFromPath` now builds a `vm.VM` and calls
  `vm.Interpret(file)` instead of driving `compiler.Compiler` directly
  (folds in the prior 2026-08-24 pending worklog's fix for the same
  stranded call sites — that entry's scope is superseded by this commit).
- `scanner_test.go` — 9 call sites renamed `tok.GetLexme()` → `tok.Lexeme()`
  (closes ISSUE-022).
- `examples/scanner_text.glox` — fixture replaced with a single malformed
  f-string case.
- `.claude/ISSUES.md` — closed ISSUE-022; opened ISSUE-023..026 for the
  defects found in this diff (see Evidence).
- `.gitignore` (new) — ignores `/glox` and `*.exe` so the local build
  binary sitting in the repo root stops showing as untracked.
- `pratt-parser.md` (new) — working notes, not reviewed as code.

## Why
Continuing the Pratt-parser buildout for expression compilation (precedence
table + prefix/infix parse functions), following the `compiler.go`/VM
plumbing fixed in the 2026-08-24 session. Value representation was
simplified from `value.Value` to a bare `float64` for now (no boxed value
type yet) to keep the constant pool and VM stack moving in step with the
in-progress `number()`/`makeConstant()` parse rule.

## How
Committing now as an explicit **work-in-progress snapshot** at the user's
request, not because the code is done — `go build ./...` currently fails
and the user chose to commit anyway rather than fix first (asked directly:
fix now vs. commit as WIP; answer was WIP). The four defects below are
tracked in ISSUES.md rather than fixed here, so the broken state is visible
and actionable in the next session rather than silently reintroduced.

## Evidence
```
$ go build ./...
internals\compiler\precedence.go:6:2: no required module provides package honnef.co/go/tools/analysis/facts/tokenfile; to add it:
	go get honnef.co/go/tools/analysis/facts/tokenfile
```
Build fails on the first error above (`precedence.go`'s bad import).
Additional defects found by inspection, not yet surfaced by the compiler
because the build doesn't get past the import error — tracked as:
- ISSUE-023 — bad staticcheck-internal import in `precedence.go:6`
- ISSUE-024 — undefined `PREC_NONE` / bad `ParseFunc` value in the `rules` table
- ISSUE-025 — syntax error (extra `)`) + undefined `rule.precedence`/`getRule` in `Binary()`
- ISSUE-026 — apparently-unused `text/template/parse` import in `compiler.go`

No tests were run against this diff — `go test ./...` will fail at the
build step for the same reason `go build ./...` does. This entry makes no
correctness claim beyond "these specific line-level diffs are what
changed."

## Stuck points
None worked through this session — the build errors above are left open
as tracked issues rather than fixed, per explicit user instruction to
commit the WIP state as-is.
