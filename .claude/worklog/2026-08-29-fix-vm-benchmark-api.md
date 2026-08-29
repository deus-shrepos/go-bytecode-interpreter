---
date: 2026-08-29
phase: vm
files: [internals/vm/vm_bench_test.go]
commit: 162150c
---

# Fix BenchmarkVM to match current NewVM/Interpret signatures

## What
- `vm_bench_test.go:9-15`: replaced manual chunk construction
  (`memory.Alloc[compiler.Chunk]`, `chunk.WriteChunk`/`WriteConstant` calls,
  `NewVM(arena, chunk, false)`, no-arg `v.Interpret()`, trailing
  `arena.Free()`) with `v := NewVM(arena, false)` and
  `v.Interpret([]byte("1.2 + 1.0"))`.
- Dropped the now-unused `compiler` import.

## Why
`go build ./...` failed on this file: `NewVM` and `Interpret` had been
changed elsewhere to `NewVM(arena, trace bool)` and
`Interpret(source []byte)` — `Interpret` now compiles from source itself
(`vm.go:73-89`) rather than taking a pre-built `*Chunk`. This benchmark was
never updated to match, so it wouldn't compile. User asked to fix it
directly since it's a test file (allowed to edit per project mentor
contract).

## How
Read `vm.go`'s current `NewVM`/`Interpret` definitions to get the real
signatures rather than guessing. Noted that `Interpret` already calls
`vm.arena.Free()` on both its error and success return paths (`vm.go:81,87`)
— the benchmark's old trailing `arena.Free()` would have been a double-free
against the new API, so it was dropped rather than kept.

## Evidence
```
$ go build ./...
(clean)

$ go test ./internals/vm/... -run xxx -bench BenchmarkVM -benchmem
BenchmarkVM-16    53916    27213 ns/op    262545 B/op    7 allocs/op
PASS
ok  	go-bytecode-interpreter/internals/vm	1.697s
```
Flagged to the user (not fixed here): this benchmark measures arena
setup/teardown + compile cost per iteration, not isolated VM dispatch-loop
cost, since a fresh 256KB arena is allocated and freed every `b.N` iteration.
