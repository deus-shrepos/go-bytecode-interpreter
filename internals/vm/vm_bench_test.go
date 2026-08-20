package vm

import (
	"go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"testing"
)

func BenchmarkVM(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		arena := memory.NewArena(1 << 18)
		chunk := memory.Alloc[compiler.Chunk](arena)
		*chunk = compiler.NewChunk(arena)
		chunk.WriteChunk(compiler.OP_CONST, 1)
		chunk.WriteConstant(1.2, 1)
		chunk.WriteChunk(compiler.OP_CONST, 1)
		chunk.WriteConstant(1.0, 1)
		chunk.WriteChunk(compiler.OP_ADD, 1)
		chunk.WriteChunk(compiler.OP_RETURN, 1)
		v := NewVM(arena, chunk, false)
		v.Interpret()
		arena.Free()
	}
}
