---
date: 2026-08-16
phase: compiler
files: [internals/compiler/opcodes.go, internals/compiler/opcode_string.go, internals/compiler/opcodes_test.go, internals/lexer/token.go]
commit: bd5c751
---

# Generate OpCode.String() via stringer, fix Token.String()'s %g/unsafe.Pointer mismatch

## What
- `internals/compiler/opcodes.go`: removed the hand-written
  `func (o OpCode) String() string` switch (8 cases + default), replaced
  with a `//go:generate stringer -type=OpCode` directive above the
  `type OpCode uint8` declaration.
- `internals/compiler/opcode_string.go` (new, generated): stringer-produced
  `String()` implementation using a packed name string + index table,
  covering `OP_RETURN` through `OP_CONST_LONG`.
- `internals/compiler/opcodes_test.go` (new): `TestOpCode_String` table-tests
  all 8 opcode constants against their expected string form.
- `internals/lexer/token.go:84`: `Token.String()`'s `Sprintf` format verb
  for `t.Start` changed from `%g` (float) to `%p` (pointer), matching
  `t.Start`'s actual `unsafe.Pointer` type. Closes ISSUE-017.

## Why
The hand-written `OpCode.String()` switch is boilerplate that drifts from
the `const` block whenever an opcode is added or reordered — nothing
enforces the two stay in sync. `go:generate stringer` derives the method
from the const declarations directly, so a missed case becomes a compile-
time array-index error (see the `_ = x[OP_NEGATE-2]` guards in the
generated file) instead of a silent "UNKNOW OP_CODE" at runtime.

The `token.go` fix was a pre-existing bug surfaced by `go vet` while
working the scanner-keyword diff (recorded in the prior
`2026-08-16-scanner-keyword-fix-and-arena-wiring` entry as ISSUE-017): since
`go test` runs `vet` as a build step, the wrong verb blocked every test in
`internals/lexer` from running at all. Fixing it here unblocks that
package's test suite as part of the same commit.

## How
Used `go generate` rather than hand-maintaining the switch or writing a
custom codegen script — stringer is the standard tool for this exact
mapping and its output includes a compile-time consistency check against
the const block, which a hand-written switch can't provide. The generated
file is checked in (not regenerated at build time) so the package builds
without requiring `stringer` as a build dependency.

For the token.go fix, `%p` was chosen over stripping the field or adding a
custom formatter, since `t.Start` is deliberately an `unsafe.Pointer` into
the source buffer (see ISSUE-005) and printing it as a pointer address is
the correct debug representation — no other part of the diff touches how
`Start` is used.

## Evidence
- `go build ./...` — succeeds.
- `go test ./internals/compiler/... ./internals/lexer/...` — both `ok`.
- `go vet ./internals/lexer/...` — no output (confirms ISSUE-017's `%g`
  mismatch is gone).
- `go test ./...` — fails only on `internals/vm` with `package
  go-bytecode-interpter/internals/compiler is not in std`, a pre-existing
  stale import from the module rename in `344b2b0`, unrelated to this diff.
  Logged as ISSUE-019; out of scope for this commit.
