package vm

import (
	"fmt"
	"go-bytecode-interpreter/internals/memory"
	"strings"
	"testing"
)

func BenchmarkVM(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		arena := memory.NewArena(1 << 18)
		v := NewVM(arena, false)
		v.Interpret([]byte("1.2 + 1.0"))
	}
}

// chainedSum builds "1.0 + 1.0 + ... + 1.0" with n terms — enough
// instructions to push Chunk.Code/Consts/Lines past their initial 64-slot
// arena capacity (chunks.go:19-21), so growth has to happen somewhere.
func chainedSum(n int) []byte {
	var sb strings.Builder
	sb.WriteString("1.0")
	for i := 1; i < n; i++ {
		sb.WriteString(" + 1.0")
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
	for _, n := range []int{10, 100, 1_000, 10_000, 100_000, 1_000_000} {
		source := chainedSum(n)
		b.Run(fmt.Sprintf("terms=%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				arena := memory.NewArena(1 << 18)
				v := NewVM(arena, false)
				v.Interpret(source)
			}
		})
	}
}
