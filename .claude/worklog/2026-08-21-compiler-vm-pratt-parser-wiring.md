---
date: 2026-08-21
phase: compiler
files: [internals/compiler/compiler.go, internals/errors/errors.go, internals/lexer/scanner.go, internals/vm/vm.go, examples/scanner_text.glox]
commit: 1b0f0bc
---

# Wire Compiler/VM for Pratt-parser expression compilation, clox-style errors

## What
- `Compiler` restructured to hold `current`/`previous` tokens, an owned
  `Scanner`, and a `*Chunk` (compiler.go:12-19), replacing the old
  arena-only struct.
- `NewCompiler` now takes `chunk *Chunk` and `source []byte` and constructs
  the scanner internally (compiler.go:21-27).
- `Compile()` changed from a token-dump loop to the start of a real parse:
  `advance()` + `expression()` + `consume(EOF, ...)` (compiler.go:29-34);
  `expression()` and `consume()` are stubs with empty bodies
  (compiler.go:44-49).
- `advance()` now loops past `ERROR` tokens, reporting each via the new
  `errors.NewError` (compiler.go:36-42).
- `errors.InterpreterError` renamed to `errors.Error`, dropped its `Line`
  field, gained a `token lexer.Token` field; `Error()` reformats clox-style:
  `[line N] Phase at 'lexeme'` / `at end` for EOF (errors.go:14-38). Added
  unexported `errorAt()` stub (errors.go:47-49).
- `vm.VM` gained an `arena *memory.Arena` field; `NewVM` dropped its
  `chunk` param (vm.go:26-33).
- `vm.Interpret` signature changed from `(source string)` (void) to
  `(source []byte) InterpretResult`: it now builds the `Chunk` and
  `Compiler` itself, compiles, frees the arena and returns
  `InterpretCompilerError` on failure, else runs and frees on completion
  (vm.go:76-92).
- `scanner.go`: whitespace-only diff (blank line removals), no logic
  change.
- `examples/scanner_text.glox`: added a second f-string example using a
  bound variable (`var x = "foo"` + `f"I am now invoking {x}"`).

## Why
First step toward real Pratt-parser-based expression compilation: moves
ownership of the compile-then-run pipeline into `VM.Interpret` and gives
`Compiler` the token/scanner state a Pratt parser needs, ahead of filling
in `expression()`/`consume()`. The `errors.Error` rework switches from a
bare line number to a token-anchored, clox-style message so later parser
errors can point at the offending lexeme.

## How
`VM.Interpret` now owns chunk/compiler construction per-call rather than
taking a pre-built chunk, so each `Interpret` call gets a fresh
arena-scoped chunk and frees the arena on both the compile-error and
success paths (vm.go:79-92) — replaces the previous split where `NewVM`
took a chunk up front. `errors.Error` takes the offending `lexer.Token`
directly instead of a raw line number, matching clox's `errorAt(Token*,...)`
so future parser errors (missing token, unexpected token) can be reported
uniformly through `Compiler.advance`'s error path.

## Evidence
```
go build ./...
# internals\repl\repl.go:23:35: not enough arguments in call to compiler.NewCompiler
#   have (*memory.Arena)
#   want (*memory.Arena, *compiler.Chunk, []byte)
# internals\repl\repl.go:24:19: too many arguments in call to compiler.Compile
#   have ([]byte)
#   want ()
# internals\vm\vm.go:79:38: not enough arguments in call to c.NewCompiler
#   have (*memory.Arena, *compiler.Chunk)
#   want (*memory.Arena, *compiler.Chunk, []byte)
# internals\vm\vm.go:83:23: too many arguments in call to compiler.Compile
#   have ([]byte)
#   want ()

go vet ./...
# internals\errors\errors.go:37:45: fmt.Sprintf format %.*s has arg &e.token.Start of wrong type *unsafe.Pointer
# internals\errors\errors.go:39:21: non-constant format string in call to fmt.Sprintf
# internals\compiler\compiler.go:43:27: fmt.Fprintf format %s has arg err of wrong type go-bytecode-interpreter/internals/errors.Error

go test ./...
FAIL	go-bytecode-interpreter/cmd/glox [build failed]
FAIL	go-bytecode-interpreter/internals/compiler [build failed]
?   	go-bytecode-interpreter/internals/debug	[no test files]
FAIL	go-bytecode-interpreter/internals/errors [build failed]
ok  	go-bytecode-interpreter/internals/lexer	(cached)
ok  	go-bytecode-interpreter/internals/memory	(cached)
FAIL	go-bytecode-interpreter/internals/repl [build failed]
?   	go-bytecode-interpreter/internals/value	[no test files]
FAIL	go-bytecode-interpreter/internals/vm [build failed]
```

## Stuck points
Committed with a known broken build (user's explicit call, not fixed here
per project contract — assistant reviews, doesn't author implementation):
- `compiler.go`'s `NewCompiler`/`Compile` signatures changed but three call
  sites were never updated to match: `vm.go:79,83`, `repl.go:23-24`, and
  `vm_bench_test.go:21-22` (which also still calls the old 3-arg `NewVM`
  and no-arg-less-`Interpret`).
- `errors.go:37` — `%.*s` is given `&e.token.Start` (a `*byte`), not a
  string; won't format the lexeme as intended.
- `compiler.go:43` — `Fprintf("%s", err)` passes `errors.Error` by value,
  but `Error()` is defined on `*errors.Error`, so the value doesn't
  satisfy the `error`/`Stringer` interface and `%s` won't call it.

Follow-up commit needed: sync the three call sites to the new signatures,
fix the two format-verb bugs, then confirm `go build ./...`, `go vet ./...`,
and `go test ./...` are clean.
