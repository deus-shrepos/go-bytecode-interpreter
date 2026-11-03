# Agenc (WIP)

A small programming language for writing programs that use AI models
safely. Model calls are typed functions, tools are capabilities passed in as
arguments, and every agent loop runs under rules set by the program, not by
the model.

This repository is Lin's implementation: a bytecode compiler and stack-based
virtual machine written in Go. It started as a port of `clox` from
*Crafting Interpreters*, and the command-line tool and example files still
use clox's `glox` name.

## Why Agenc

Agent programs today are mostly ordinary code wrapped around a model call.
What the model may do, which tools it can reach, and when a loop has to stop
are decided by convention, prompt text, or framework settings, not by the
program itself. Lin puts those decisions into the language, where the
compiler and runtime can check them:

- **Model calls are typed functions.** A call to a model has a declared input
  and output type, like any other function. The program gets a value of that
  type or an error, never unchecked text.
- **Tools are capabilities passed as arguments.** Code can use a tool only if
  it was given one. What an agent can do is visible in its function
  signatures, and it can't reach anything it wasn't handed.
- **The program sets the rules for every agent loop.** Limits on steps,
  cost, and time, and the conditions for stopping, belong to the program.
  The model works inside them; it can't change them.

None of this exists yet. These are the design goals the implementation is
working toward.

## Why build it this way

The implementation is also a learning project: working through the whole
pipeline from source bytes to executed instructions, in a language with a
real garbage collector and real profiling tools, so each design decision can
be tried, tested, and measured:

- how source text becomes tokens, and what a token should and shouldn't own
- how a parser builds a tree that later passes can walk
- how a tree is lowered to bytecode, and what a VM does with it
- where memory goes, and when an arena allocator actually helps

The agent-specific parts (types for model calls, capabilities, loop rules)
depend on that foundation, so they come after the core language works.

## Status

The project is midway through a restructure. It is moving from clox's
single-pass design (the compiler pulls tokens and emits bytecode directly) to
separate phases:

```
source bytes → tokens → AST → bytecode → VM
```

| Package | State |
|---|---|
| `internals/token` | Done. A token is a 12-byte `{Start, End, Kind}` span into the source. Line and column are computed from the offset only when an error is reported. |
| `internals/lexer` | Done. One pass over the whole file produces tokens, line-start offsets, and diagnostics. Handles strings, escapes, comments, and f-strings with nested interpolation. Errors don't stop scanning. |
| `internals/ast` | Started. A flat node array whose children are indices, not pointers. |
| `internals/parser` | In progress. A Pratt parser that will build AST nodes. |
| `internals/compiler` | The original clox-style single-pass compiler for arithmetic expressions. It will be replaced by lowering from the AST. |
| `internals/vm` | A stack VM running constants, negation, and the four arithmetic operators. |
| `internals/memory` | An arena allocator, with benchmarks comparing it to the Go heap. |

Because the compiler hasn't been moved to the new token API yet,
`go build ./...` currently fails. The `token` and `lexer` packages build and
pass their tests on their own.

## How it's developed

The project's author writes all the implementation code. An AI assistant
(Claude, through Claude Code) acts as a reviewer and tutor sometimes. It explains
concepts, reviews changes, traces bugs by hand, and writes tests when asked,
but it does not write the language implementation. The rules are in
`.claude/CLAUDE.md` and `.claude/output-styles/mentor.md`.

A few rules shape the code:

- **Correct first, then fast.** No optimization goes in before the code it
  changes is tested.
- **No performance claim without a measurement.** "This is faster" stays a
  hypothesis until `go test -bench`, `benchstat`, or `pprof` shows it.
- **Concrete over abstract.** A `switch` over node kinds is preferred to a
  visitor framework, and a function earns an abstraction only after the same
  code has been written three times.
- **Every commit has a work-log entry** in `.claude/worklog/` recording what
  changed, why, how, and the test or benchmark output that backs it up.
- **Bugs are tracked as GitHub Issues.** Commits reference them with
  `Fixes #N`. Earlier bugs are recorded in `.claude/ISSUES.md`, which is kept
  as history.

## Building and running

Requires Go 1.27 or later.

```sh
go build -o glox ./cmd/glox

./glox compile examples/simple_maths.glox                  # compile only
./glox compile --trace-compiler examples/simple_maths.glox # with a compiler trace
./glox run examples/simple_maths.glox                      # compile and execute
```

These commands go through the old compiler, so they won't build until the
compiler is migrated (see Status).

## Tests and measurement

```sh
go test ./internals/token ./internals/lexer    # packages that build today
go test ./...                                  # everything, once the build is fixed
```

The scanner tests check the complete token stream for each input, and run a
set of invariants on every input: exactly one EOF token at the end, spans in
bounds and non-overlapping, one diagnostic per error token.

The `Makefile` wraps the performance tooling:

| Target | What it does |
|---|---|
| `make bench-stress` | VM stress benchmark from 10 to 1,000,000 terms, with allocation stats |
| `make bench-baseline` / `make bench-compare` | Save a benchmark baseline, then compare later runs against it with `benchstat` |
| `make pprof-heap` / `make pprof-view` | Capture a heap profile and open it in pprof's web UI |
| `make memprobe-arena-gc` | Arena vs Go heap: GC cycles, pause time, allocation counts |
| `make escape-analysis` | Show the Go compiler's escape-analysis decisions for `memory` and `vm` |

## Layout

```
cmd/glox/            command-line entry point (compile, run)
internals/token/     token kinds, spans, line/column lookup
internals/lexer/     scanner
internals/ast/       AST node layout
internals/parser/    Pratt parser (in progress)
internals/compiler/  bytecode chunks, opcodes, current single-pass compiler
internals/vm/        stack VM and its benchmarks
internals/memory/    arena allocator
examples/            small .glox programs
.claude/             assistant configuration, work log, issue history
```

