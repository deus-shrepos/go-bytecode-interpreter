package memory

import (
	"testing"
	"unsafe"
)

// Realistic interpreter data structures used across all benchmarks.

type vmValue struct {
	tag  uint8
	_    [7]byte
	data int64
}

type vmInstr struct {
	op      uint8
	a, b, c uint8
	imm     int32
}

type callFrame struct {
	ip     int32
	argc   int32
	locals []vmValue
}

// =============================================================================
// Scalar: single value per iteration.
// Shows raw bump-pointer speed vs heap allocation for the smallest unit of work.
// =============================================================================

func BenchmarkHeapScalar(b *testing.B) {
	var sink *vmValue
	for i := 0; i < b.N; i++ {
		v := &vmValue{tag: uint8(i), data: int64(i)}
		sink = v
	}
	_ = sink
}

func BenchmarkArenaScalar(b *testing.B) {
	const batch = 1 << 13
	a := NewArena(batch * int(unsafe.Sizeof(vmValue{})))
	b.ResetTimer()
	var sink *vmValue
	for i := 0; i < b.N; i++ {
		if i > 0 && i%batch == 0 {
			a.Reset()
		}
		v := Alloc[vmValue](a)
		v.tag = uint8(i)
		v.data = int64(i)
		sink = v
	}
	_ = sink
}

// =============================================================================
// Locals slice: one slice of N values per iteration.
// Represents allocating a function's local variable table.
// =============================================================================

func BenchmarkHeapLocals(b *testing.B) {
	const n = 16
	var sink []vmValue
	for i := 0; i < b.N; i++ {
		locals := make([]vmValue, n)
		locals[0] = vmValue{tag: 1, data: int64(i)}
		sink = locals
	}
	_ = sink
}

func BenchmarkArenaLocals(b *testing.B) {
	const n = 16
	const batch = 512
	a := NewArena(batch * n * int(unsafe.Sizeof(vmValue{})))
	b.ResetTimer()
	var sink []vmValue
	for i := 0; i < b.N; i++ {
		if i > 0 && i%batch == 0 {
			a.Reset()
		}
		locals := AllocSlice[vmValue](a, n)
		locals[0] = vmValue{tag: 1, data: int64(i)}
		sink = locals
	}
	_ = sink
}

// =============================================================================
// Call frame: frame struct + locals slice, all written to.
// One heap alloc for the struct + one for the slice vs two bump-pointer increments.
// =============================================================================

func BenchmarkHeapCallFrame(b *testing.B) {
	var sink *callFrame
	for i := 0; i < b.N; i++ {
		f := &callFrame{
			ip:     int32(i),
			argc:   4,
			locals: make([]vmValue, 4),
		}
		for j := range f.locals {
			f.locals[j] = vmValue{tag: uint8(j), data: int64(i + j)}
		}
		sink = f
	}
	_ = sink
}

func BenchmarkArenaCallFrame(b *testing.B) {
	frameBytes := int(unsafe.Sizeof(callFrame{})) + 4*int(unsafe.Sizeof(vmValue{}))
	const batch = 512
	a := NewArena(batch * frameBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i > 0 && i%batch == 0 {
			a.Reset()
		}
		f := Alloc[callFrame](a)
		f.ip = int32(i)
		f.argc = 4
		f.locals = AllocSlice[vmValue](a, 4)
		for j := range f.locals {
			f.locals[j] = vmValue{tag: uint8(j), data: int64(i + j)}
		}
		_ = f
	}
}

// =============================================================================
// Deep call stack: N nested function calls, each with its own frame and locals.
// This is where Mark/Release dominates — the entire stack collapses in one call
// vs N GC-eligible heap allocations per frame level.
// =============================================================================

const stackDepth = 32

func BenchmarkHeapDeepCallStack(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var frames [stackDepth]*callFrame
		for j := range stackDepth {
			f := &callFrame{
				ip:     int32(j),
				argc:   int32(j % 4),
				locals: make([]vmValue, j%8+1),
			}
			for k := range f.locals {
				f.locals[k] = vmValue{tag: uint8(k), data: int64(j*10 + k)}
			}
			frames[j] = f
		}
		_ = frames
	}
}

func BenchmarkArenaDeepCallStack(b *testing.B) {
	// Worst-case: 8 locals per frame.
	frameBytes := int(unsafe.Sizeof(callFrame{})) + 8*int(unsafe.Sizeof(vmValue{}))
	a := NewArena(stackDepth * frameBytes)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mark := a.Mark()
		for j := range stackDepth {
			f := Alloc[callFrame](a)
			f.ip = int32(j)
			f.argc = int32(j % 4)
			f.locals = AllocSlice[vmValue](a, j%8+1)
			for k := range f.locals {
				f.locals[k] = vmValue{tag: uint8(k), data: int64(j*10 + k)}
			}
			_ = f
		}
		a.Release(mark)
	}
}

// =============================================================================
// String allocation: interning identifiers from source code.
// In a language runtime, every identifier, keyword, and string literal
// must be copied out of the source buffer. AllocString pins them in the arena.
// =============================================================================

var identifiers = [...]string{
	"counter", "index", "value", "result", "temp",
	"accumulator", "iterator", "length", "capacity", "offset",
	"source", "target", "buffer", "stream", "handler",
}

func BenchmarkHeapString(b *testing.B) {
	var sink string
	for i := 0; i < b.N; i++ {
		// string([]byte(s)) is what a lexer does: copy bytes out of a buffer.
		sink = string([]byte(identifiers[i%len(identifiers)]))
	}
	_ = sink
}

func BenchmarkArenaString(b *testing.B) {
	const batch = 4096
	a := NewArena(batch * 16)
	b.ResetTimer()
	var sink string
	for i := 0; i < b.N; i++ {
		if i > 0 && i%batch == 0 {
			a.Reset()
		}
		sink = AllocString(a, identifiers[i%len(identifiers)])
	}
	_ = sink
}

// =============================================================================
// Reset pattern: allocate N instructions, execute them, reset.
// Simulates one bytecode function invocation. The heap version must wait for GC
// to reclaim the per-call allocations; the arena resets in a single clear.
// =============================================================================

const instrPerCall = 32

func BenchmarkHeapResetPattern(b *testing.B) {
	var sink []*vmInstr
	for i := 0; i < b.N; i++ {
		instrs := make([]*vmInstr, instrPerCall)
		for j := range instrs {
			instrs[j] = &vmInstr{op: uint8(j), imm: int32(j)}
		}
		sink = instrs
	}
	_ = sink
}

func BenchmarkArenaResetPattern(b *testing.B) {
	a := NewArena(instrPerCall * int(unsafe.Sizeof(vmInstr{})))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for j := range instrPerCall {
			instr := Alloc[vmInstr](a)
			instr.op = uint8(j)
			instr.imm = int32(j)
		}
		a.Reset()
	}
}
