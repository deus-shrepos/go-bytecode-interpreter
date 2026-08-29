package vm

import (
	"go-bytecode-interpreter/internals/memory"
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
