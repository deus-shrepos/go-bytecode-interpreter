---
date: 2026-08-22
phase: compiler
files: [internals/compiler/compiler.go, internals/errors/errors.go, internals/lexer/token.go]
commit: pending
---

# Add panic-mode error recovery to the compiler

## What
- `compiler.go:12-19` — `Compiler` gains `hadError`/`panicMode bool` fields.
- `compiler.go:34` — `Compile()` returns `!c.hadError` instead of a hardcoded
  `false`.
- `compiler.go:44-53` — `advance()`'s error-token loop now checks
  `c.panicMode` and returns early (resync) instead of unconditionally
  reporting every `ERROR` token; the report itself moved out of `advance()`
  into the new `reportError()`.
- `compiler.go:59-65` — new `consume(tokeType, message)`: advances on a type
  match, else calls `reportError(message)`.
- `compiler.go:67-69` — new `EmitByteCode(byteCode byte)` stub, empty body.
- `compiler.go:71-79` — new `reportError(message)`: sets `panicMode` and
  `hadError`, prints via `errors.NewError(...)` to stderr.
- `errors.go:32-40` — `Error()` switches on `token.Type` (`EOF`/`ERROR`/
  default) instead of an if/else chain; the default case now formats via
  the new `Token.Lexeme()` instead of the old `%.*s` + pointer construct.
- `errors.go:46-48` — `unwrap()` renamed to exported `Unwrap()`.
- `errors.go` — dead `errorAt()` stub removed.
- `token.go:84` — `GetLexme` renamed to `Lexeme`.

## Why
Implements panic-mode synchronization (clox's parser error-recovery
strategy): once one syntax error is reported, further `ERROR` tokens
encountered while resyncing are swallowed instead of each producing its own
spurious message. This is the parser error-recovery phase-checklist item.

## How
`panicMode` gates `advance()`'s error path — set by `reportError()`, checked
before `advance()` reports a scanner `ERROR` token again. `reportError()`
centralizes what used to be an inline `Fprintf` in `advance()`, and is now
the single place that both flips the error flags and prints, so `consume()`
(the first caller of `reportError()` outside the scanner-error path) gets
the same panic-mode gating for parser-level "expected token" errors for
free.

## Evidence
```
$ go build ./...
internals/repl/repl.go:23:35: not enough arguments in call to compiler.NewCompiler
        have (*memory.Arena)
        want (*memory.Arena, *compiler.Chunk, []byte)
internals/repl/repl.go:24:19: too many arguments in call to compiler.Compile
        have ([]byte)
        want ()
internals/vm/vm.go:79:38: not enough arguments in call to c.NewCompiler
        have (*memory.Arena, *compiler.Chunk)
        want (*memory.Arena, *compiler.Chunk, []byte)
internals/vm/vm.go:83:23: too many arguments in call to compiler.Compile
        have ([]byte)
        want ()

$ go vet ./...
vet: internals/lexer/scanner_test.go:119:18: tok.GetLexme undefined (type lexer.Token has no field or method GetLexme)
internals/errors/errors.go:43:21: non-constant format string in call to fmt.Sprintf
internals/compiler/compiler.go:76:4: fmt.Fprintf format %s has arg errors.NewError(c.current, errors.CompilePhaseError, message) of wrong type go-bytecode-interpreter/internals/errors.Error
vet: internals/vm/vm.go:79:44: not enough arguments in call to c.NewCompiler

$ go test ./...
FAIL    go-bytecode-interpreter/cmd/glox [build failed]
FAIL    go-bytecode-interpreter/internals/compiler [build failed]
?       go-bytecode-interpreter/internals/debug        [no test files]
FAIL    go-bytecode-interpreter/internals/errors [build failed]
FAIL    go-bytecode-interpreter/internals/lexer [build failed]
ok      go-bytecode-interpreter/internals/memory       (cached)
FAIL    go-bytecode-interpreter/internals/repl [build failed]
?       go-bytecode-interpreter/internals/value        [no test files]
FAIL    go-bytecode-interpreter/internals/vm [build failed]
```

## Stuck points
Committed with known failures, user's explicit call ("commit as-is, record
failures") — not fixed here per project contract (assistant reviews,
doesn't author implementation):
- `errors.go:43` `fmt.Sprintf(errorString.String())` — non-constant format
  string, vet-flagged.
- `compiler.go:76` `Fprintf(os.Stderr, "%s", errors.NewError(...))` —
  `NewError` returns `Error` by value, `Error()` has a pointer receiver, so
  `%s` never actually invokes it; vet-flagged. This is the same bug the
  2026-08-21 entry described in prose but never gave a tracked id — now
  ISSUE-021.
- `token.go`'s `GetLexme`→`Lexeme` rename breaks `internals/lexer/scanner_test.go`
  (9 call sites still reference `GetLexme`) — new, caused by this diff, test
  file itself not part of it. Now ISSUE-022.
- `internals/repl` and `internals/vm` still fail to build on pre-existing
  signature mismatches (tracked in the 2026-08-21 entry's stuck points,
  unrelated to this diff).
