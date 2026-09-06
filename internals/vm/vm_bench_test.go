package vm

import (
	"fmt"
	c "go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"runtime"
	"strings"
	"testing"
	"unsafe"
)

func BenchmarkVM(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		arena := memory.NewArena(256)
		v := NewVM(arena, false)
		v.Interpret([]byte("1.2 + 1.0"))
	}
}

// chainedSum builds "1.0 + 1.0 + ... + 1.0" with n terms — enough
// instructions to push Chunk.Code/Consts/Lines past their initial 64-slot
// arena capacity (chunks.go:19-21), so growth has to happen somewhere.
func chainedSum(n int) []byte {
	var sb strings.Builder
	sb.WriteString("100.0")
	for i := 1; i < n; i++ {
		sb.WriteString(" + 200.0")
	}
	return []byte(sb.String())
}

// BenchmarkVM_Stress compiles and runs chained-addition expressions of
// increasing size, one full compile+run per b.N iteration (same
// single-shot pattern as BenchmarkVM — Interpret frees its own arena at the
// end, so there's no reuse-across-iterations variant here). Compare
// allocs/op and B/op across sizes with benchstat to see where growth
// stops being free.
func BenchmarkVM_Stress(b *testing.B) {
	for _, n := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000} {
		source := chainedSum(n)
		b.Run(fmt.Sprintf("terms=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				arena := memory.NewArena(10)
				v := NewVM(arena, false)
				v.Interpret(source)
			}
		})
	}
}

// BenchmarkVM_Stress_Phases replicates Interpret's own compile-then-run
// sequence (vm.go:73-89) instead of calling Interpret directly, so
// runtime.ReadMemStats can be snapshotted between phases. TotalAlloc is
// cumulative bytes allocated (not live heap size), so each diff is exactly
// the bytes allocated during that phase — this attributes Go-heap growth to
// compile (Chunk.Code/Consts/Lines outgrowing their 64-slot arena capacity,
// chunks.go:19-21) vs run (vm.Stack growth via Push's append, vm.go:102-104)
// instead of lumping both into one Interpret-wide alloc count.
func BenchmarkVM_Stress_Phases(b *testing.B) {
	for _, n := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000, 10_000_000} {
		source := chainedSum(n)
		b.Run(fmt.Sprintf("terms=%d", n), func(b *testing.B) {
			var compileBytes, runBytes uint64
			for i := 0; i < b.N; i++ {
				arena := memory.NewArena(10)
				v := NewVM(arena, false)

				var m0, m1, m2 runtime.MemStats
				runtime.ReadMemStats(&m0)

				chunk := c.NewChunk(arena)
				compiler := c.NewCompiler(arena, &chunk, source)
				if !compiler.Compile() {
					arena.Free()
					b.Fatal("compile failed")
				}
				runtime.ReadMemStats(&m1)

				v.Chunk = &chunk
				v.ip = (*uint8)(unsafe.Pointer(&v.Chunk.Code[0]))
				v.Run()
				runtime.ReadMemStats(&m2)

				compileBytes += m1.TotalAlloc - m0.TotalAlloc
				runBytes += m2.TotalAlloc - m1.TotalAlloc
				arena.Free()
			}
			b.ReportMetric(float64(compileBytes)/float64(b.N), "compile-B/op")
			b.ReportMetric(float64(runBytes)/float64(b.N), "run-B/op")
		})
	}
}
