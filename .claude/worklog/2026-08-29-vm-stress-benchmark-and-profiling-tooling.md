---
date: 2026-08-29
phase: vm
files: [internals/vm/vm_bench_test.go, Makefile, .gitignore]
commit: 382d157
---

# Add a VM stress benchmark and reusable heap-profiling Makefile targets

## What
- `vm_bench_test.go:19-29`: added `chainedSum(n int) []byte`, generating
  `"1.0 + 1.0 + ... + 1.0"` with `n` terms.
- `vm_bench_test.go:37-49`: added `BenchmarkVM_Stress`, table-driven over
  `n ∈ {10, 100, 1_000, 10_000, 100_000, 1_000_000}`, one compile+run per
  `b.N` iteration (same single-shot arena-per-iteration shape as the
  existing `BenchmarkVM`).
- `Makefile`: added `bench-stress`, `gctrace-stress`, `pprof-heap`,
  `pprof-view`, `clean-pprof` targets, plus `VM_PKG`, `PPROF_DIR`,
  `PPROF_PORT`, `STRESS_BENCHTIME` variables. `pprof-heap` builds a pinned
  `go test -c` binary into `.pprof/` and captures a full-resolution
  (`-test.memprofilerate=1`) heap profile; `pprof-view` serves it via
  `go tool pprof -http`.
- `.gitignore`: added `*.test` and `/.pprof/` — the pinned test binary and
  captured profiles shouldn't be committed.
- Removed a stray `internals/vm/vm.test` binary that had been accidentally
  written into the package directory by an earlier ad-hoc `go test -c`
  invocation during this session, before the `.pprof/`-scoped Makefile
  targets existed.

## Why
The original `BenchmarkVM` only compiles/runs a 9-byte source
(`"1.2 + 1.0"`), so `Chunk.Code`/`Consts`/`Lines` never leave their initial
64-slot arena capacity (`chunks.go:19-21`) — it can't show what happens once
`WriteChunk`/`AddConstant`'s plain `append` calls have to grow past that.
User asked for a benchmark that stress-tests the arena at realistic scale and
a reusable, repeatable way to visualize the resulting allocation profile,
rather than re-typing the same `go test -bench`/`go tool pprof` invocations
by hand each time.

## How
`chainedSum` was chosen over a recursive/nested expression generator because
`+` is left-associative and its `binary()` parse loop (not recursion) handles
chained same-precedence operators — confirmed in-session by tracing
`1-2-3`'s call stack — so arbitrarily many chained terms stay at constant
parser recursion depth; no stack-overflow risk at `n=1_000_000`.

`pprof-heap` builds the test binary once via `go test -c` rather than
piping straight through `go test -bench ... -memprofile=...`, so the binary
persists at a stable path (`.pprof/vm.test`) for `go tool pprof` to reopen
later — `go test`'s normal temp-binary path gets cleaned up immediately
after the run, which broke `go tool pprof -http` in-session (it needs the
binary alongside the profile to resolve symbols). `pprof-view` uses a fixed
port (`6060`) rather than `:0` — `go tool pprof -http=:0`'s browser
auto-launch failed in this sandboxed environment (no display), and even
without a browser it printed the literal string `http://localhost:0`
instead of the resolved port; `-no_browser` plus a fixed port sidesteps
both problems and gives a stable, reusable URL.

## Evidence
```
$ make bench-stress
BenchmarkVM_Stress/terms=10-16         40952    26041 ns/op    262529 B/op    6 allocs/op
BenchmarkVM_Stress/terms=100-16        29794    40031 ns/op    264449 B/op   10 allocs/op
BenchmarkVM_Stress/terms=1000-16        6482   155528 ns/op    295040 B/op   18 allocs/op
BenchmarkVM_Stress/terms=10000-16       1164  1206857 ns/op    773377 B/op   34 allocs/op
BenchmarkVM_Stress/terms=100000-16       100 11428784 ns/op   5901597 B/op   51 allocs/op
BenchmarkVM_Stress/terms=1000000-16       12 88793433 ns/op  58731822 B/op   71 allocs/op
PASS

$ make pprof-heap   # writes .pprof/vm.test + .pprof/heap.prof, exits 0
$ curl -s -o /dev/null -w "%{http_code}" http://localhost:6060/   # via `make pprof-view`
307   # confirms the web UI is live and redirecting to its default view
```
`go build ./...`: clean. `go test ./...`: all packages pass (cached, no new
failures). `go vet ./...`: same three pre-existing warnings as before this
diff (`errors.go:43`, `compiler.go:176`, `repl.go:19`) — none touched here.

## Stuck points
`go tool pprof -http=:0` (auto-assigned port) silently misbehaved in this
sandbox: it printed `Serving web UI on http://localhost:0` — the literal
port `0`, not the real bound port — and its background browser-launch
attempt threw unrelated Chromium/GLib errors before the process exited.
Diagnosed by `ss -ltnp | grep pprof` to find the actual bound port by PID
rather than trusting the printed URL. Switched the Makefile target to a
fixed port (`6060`) with `-no_browser` to avoid the whole class of problem
going forward.
