---
date: 2026-08-29
phase: compiler
files: [internals/compiler/compiler.go, internals/compiler/precedence.go]
commit: pending
---

# Wire up Pratt-parser precedence climbing and fix constant-emission double-write

## What
- `precedence.go`: broke a Go package-init cycle (`rules` var → method value
  `binary` → `binary`'s body reads `rules`) by moving the `rules` map literal
  out of a `var = ...` initializer into `initRules()`, invoked via
  `var _ = initRules()`. Also fixed `ParseFunc` (`*func()` → `func(*Compiler)`)
  and a stray import (`honnef.co/go/tools/...` → nothing needed).
- `compiler.go:63`: `expression()` now calls `c.parsePrecedence(PrecAssignment)`
  instead of returning immediately.
- `compiler.go:101-112`: implemented `parsePrecedence` — advance, run the
  prefix rule, then loop while `prec <= rules[current].prec` running infix
  rules. This is the core precedence-climbing loop.
- `compiler.go:128-133`: fixed `Binary` → `binary` (was unexported-breaking
  capitalization + referenced a nonexistent `getRule`/`rule.precedence`
  field); now reads `rules[operatorType].prec` and recurses at `prec + 1`
  (left-associative).
- `compiler.go:150-158`: `number()` was calling `emitByteCode` twice per
  constant — once explicitly, once as a side effect already performed inside
  `Chunk.WriteConstant` (which appends the operand byte when
  `lastByteCode() == OP_CONST`). Removed the redundant explicit emit; `number`
  now just emits `OP_CONST` then calls `makeConstant`, which owns writing the
  operand.
- `compiler.go:79-83`: `endCompiler` now always emits a real `OP_RETURN`
  (previously the chunk had no trailing return at all).

## Why
Build was broken (`WIP: Pratt-parser scaffolding` commit) with a Go
init-cycle compile error and a typo'd struct field. Once those cleared,
compiling `100 + 200` produced a corrupted chunk — printing the raw struct
showed duplicate `OP_CONST`/`OP_RETURN`-looking entries with only 2 real
constants, which turned out to be `number()`'s constant index written twice
per literal.

## How
Traced the byte stream by hand against `Chunk.WriteConstant`'s
`lastByteCode()`-driven side effect: since `number()` already emits
`OP_CONST` before calling `makeConstant`, `WriteConstant` sees `OP_CONST` as
the last-written byte and auto-appends the operand — so `number()`'s own
second `emitByteCode` call duplicated it. Confirmed by walking
`DisassembleInstruction`'s stride over the corrupted stream: it desynced
after the second constant and read a real opcode's enum value as a
constant-pool index, panicking on an out-of-range `Consts` access. Rejected
patching the disassembler to be more defensive — the actual bug was upstream
duplication, not disassembly logic, so fixed at the source (`number()`).

Left-vs-right associativity: `binary()` recurses at `rule.prec + 1`
(left-assoc for `+ - * /`), matching Crafting Interpreters §17.

## Evidence
Verified via an ad-hoc scratch program (`go run`, not committed) compiling
`"100 + 200"` and printing both the raw struct and `DisassembleChunk`:
```
Raw struct: {Lines:[{Line:1 Count:6}] Code:[OP_CONST OP_RETURN OP_CONST OP_CONST OP_ADD OP_RETURN] Consts:[100 200]}

=== 100 + 200 ===
0000    1 OP_CONST            0 '100'
0002    | OP_CONST            1 '200'
0004    | OP_ADD
0005    | OP_RETURN
```
6 code entries, exactly 4 real instructions (2×`OP_CONST`+operand, `OP_ADD`,
`OP_RETURN`), no duplication, no disassembler crash.

`go build ./...`: clean. `go test ./...`: all existing packages pass
(`internals/compiler` cached ok). `go vet ./...` reports pre-existing,
unrelated warnings only (`errors.go`, `repl.go`) — none touched by this diff.

## Stuck points
The corrupted-chunk symptom (duplicate-looking opcodes) was initially
confusable with a second, separate issue: two distinct opcodes
(`OP_RETURN=0`, `OP_CONST=1`) collide with small constant-pool indices when
printed via the shared `OpCode` Stringer, since operand bytes and instruction
bytes share one `[]OpCode` slice with no tag distinguishing them. That
collision is real and permanent (not a bug to fix) — the double-emission was
a separate, actual bug layered on top of it. Disassembling via
`DisassembleChunk`'s stride-aware walk (not raw struct printing) is the only
reliable way to tell them apart.
