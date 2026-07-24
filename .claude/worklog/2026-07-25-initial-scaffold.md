---
date: 2026-07-25
phase: lexer, compiler, vm, memory, value, tooling
files: [cmd/glox/main.go, internals/lexer/scanner.go, internals/lexer/token.go, internals/compiler/chunks.go, internals/compiler/compiler.go, internals/compiler/opcodes.go, internals/vm/vm.go, internals/vm/operations.go, internals/memory/arena.go, internals/value/value.go, internals/errors/errors.go, internals/debug/debug.go, internals/repl/repl.go, Makefile, go.mod]
commit: 4abb765
---

# Add initial project scaffold: lexer, compiler, VM, arena allocator

## What
- `cmd/glox/main.go` — CLI entrypoint, dispatches to REPL or file mode.
- `internals/lexer/{scanner.go,token.go}` — scanner/token types (not yet
  advancing past the first byte — see Stuck points).
- `internals/compiler/{chunks.go,compiler.go,opcodes.go}` — `Chunk`
  bytecode buffer, constant pool, opcode set including `OP_CONST_LONG`
  (3-byte operand form), and the compiler entrypoint.
- `internals/vm/{vm.go,operations.go}` — bytecode dispatch loop and
  arithmetic ops for `OP_RETURN`, `OP_CONST`, `OP_NEGATE`, `OP_ADD`,
  `OP_DIVIDE`, `OP_MULTIPLY`, `OP_SUBTRACT`.
- `internals/memory/arena.go` — bump-allocator arena (`arena_test.go`,
  `arena_bench_test.go` included).
- `internals/value/value.go`, `internals/errors/errors.go`,
  `internals/debug/debug.go`, `internals/repl/repl.go` — value
  representation, error type, disassembler, REPL loop.
- `Makefile`, `go.mod` — build/test/bench/vet targets.

## Why
First commit for the project — nothing was under version control before
this. Goal was to get the initial scaffold into git history as a snapshot
before continuing implementation, not to land a working build.

## How
Committed as-is rather than waiting for fixes, per explicit instruction.
Before committing, ran a correctness pass (via the project's `diff-review`
agent, since there's no prior commit to diff against — every file was
read in full) against this project's phase checklists. No implementation
changes were made as part of this pass or this commit — per this
project's mentor-only contract, bugs are recorded here for the human to
fix, not patched by the assistant. `.claude/settings.local.json` was
correctly excluded by the user's global gitignore
(`~/.config/git/ignore`) and left out of the commit.

## Evidence
`go build ./...` / `go vet ./...` / `go test ./...` as committed:
```
internals/compiler/compiler.go:14:2: declared and not used: scanner
FAIL	go-bytecode-interpter/cmd/glox [build failed]
FAIL	go-bytecode-interpter/internals/compiler [build failed]
FAIL	go-bytecode-interpter/internals/debug [build failed]
ok  	go-bytecode-interpter/internals/memory	0.002s
FAIL	go-bytecode-interpter/internals/repl [build failed]
FAIL	go-bytecode-interpter/internals/vm [build failed]
```
Only `internals/memory` builds and passes tests as committed; every other
package fails to build because of the unused `scanner` in
`compiler.go:14`.

## Stuck points
Repo does not build as of this commit. Known issues surfaced by review,
left unfixed per the mentor-only contract:
- `internals/compiler/compiler.go:14` — `scanner` declared, never used;
  blocks `go build ./...` for every downstream package.
- `internals/vm/vm.go` dispatch loop has no `case` for `OP_CONST_LONG`,
  though the compiler/disassembler both know the opcode — instruction
  stream would desync if it's ever emitted.
- `internals/compiler/chunks.go:77` — `LoadLongConst`'s `b<<8`/`c<<16`
  overflow inside `uint8` arithmetic before the OR (flagged directly by
  `go vet`); `WriteConstant` also truncates the constant-pool index to
  one byte when the caller chose `OP_CONST`.
- `internals/lexer/scanner.go` — `start`/`current` are typed `byte` (a
  character, not a source offset); there's no index into `source`, so
  the scanner cannot advance past byte 0, and `Token.line` is never set.
- `internals/repl/repl.go` builds an error with `fmt.Errorf` but never
  surfaces it; `cmd/glox/main.go` indexes `os.Args[1]` with no length
  check.
- `internals/value/value.go`'s `NewValueArray` pre-fills 8 zero `Value`s
  but starts `Count` at 0 (currently dead code, unused elsewhere).
- `internals/vm/vm_bench_test.go:22` calls `v.Interpret()` with no
  argument; doesn't compile against `Interpret(source string)`.
