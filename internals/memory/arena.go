package memory

import (
	"fmt"
	"unsafe"
)

// chunk is one node in the linked list of memory blocks.
// New chunks are prepended; a.current always points to the newest.
type chunk struct {
	buf  []byte
	next *chunk
}

// Arena is a bump-pointer allocator backed by a linked list of chunks.
// Fast path: align offset, bump by size, return pointer — no per-object overhead.
// Slow path: grow into a new chunk when the current one is full.
type Arena struct {
	base     unsafe.Pointer // pointer to the first byte of current chunk's buffer
	end      uintptr        // length of current chunk — the offset ceiling
	offset   uintptr        // next free byte within the current chunk
	current  *chunk
	chunkCap int // capacity to use when the next grow fires
	freed    bool
}

// Mark records the arena state so it can be restored with Release.
// Use it to scope allocations to a lifetime — e.g. a function call frame:
//
//	mark := scratch.Mark()
//	defer scratch.Release(mark)
type Mark struct {
	base   unsafe.Pointer
	offset uintptr
	chunk  *chunk
}

type ArenaStats struct {
	TotalChunks int
	TotalBytes  int
}

func (as ArenaStats) String() string {
	return fmt.Sprintf("{TotalChunks=%d, TotalBytes=%d}", as.TotalChunks, as.TotalBytes)
}

// NewArena creates an arena with the given initial capacity.
// Size it to fit your typical workload to avoid early grows:
//
//	permanent: arena.NewArena(1 << 20) // 1MB — bytecode, constants, interned strings
//	scratch:   arena.NewArena(1 << 16) // 64KB — call frames, temporaries
func NewArena(initialCap int) *Arena {
	headChunk := &chunk{buf: make([]byte, initialCap), next: nil}
	arena := &Arena{
		current:  headChunk,
		freed:    false,
		chunkCap: initialCap,
	}
	arena.base = unsafe.Pointer(&headChunk.buf[0])
	arena.end = uintptr(initialCap)
	return arena
}

// alloc is the hot path: align the offset, bump it forward by size, return pointer.
// Kept under the inliner cost budget by pushing the slow path into slowAlloc.
func (a *Arena) alloc(size, alignment uintptr) unsafe.Pointer {
	start := align(a.offset, alignment)
	newPtr := size + start
	if newPtr < a.end {
		a.offset = newPtr
		return unsafe.Add(a.base, start)
	}
	return a.slowAlloc(size)
}

// slowAlloc handles chunk overflow. Marked noinline so its body does not inflate
// alloc's inliner cost — this keeps the fast path inlinable end-to-end.
//
//go:noinline
func (a *Arena) slowAlloc(size uintptr) unsafe.Pointer {
	if a.freed {
		panic("arena: cannot allocate freed arena")
	}
	a.grow(int(size))
	a.offset = size
	return a.base
}

// grow doubles the chunk capacity (or uses minSize if larger) and prepends a new chunk.
// The old chunk stays reachable via next, so pointers into it remain valid.
//
//go:noinline
func (a *Arena) grow(minSize int) {
	chunkSize := max(a.chunkCap*2, minSize)
	a.chunkCap = chunkSize
	newChunk := &chunk{buf: make([]byte, chunkSize), next: a.current}
	a.current = newChunk
	a.base = unsafe.Pointer(&newChunk.buf[0])
	a.end = uintptr(chunkSize)
	a.offset = 0
}

// Reset drops all grown chunks, rewinds the offset to zero, and zeroes only
// the bytes that were written. The retained chunk keeps its backing memory
// so the next allocation cycle costs nothing.
func (a *Arena) Reset() {
	if a.freed {
		panic("arena: cannot reset a freed arena")
	}
	used := a.offset
	a.current.next = nil
	a.offset = 0
	a.base = unsafe.Pointer(&a.current.buf[0])
	a.end = uintptr(len(a.current.buf))
	clear(a.current.buf[:used])
}

// Free walks the chunk chain and nils each node so the GC can reclaim
// the backing buffers. The arena must not be used after this call.
func (a *Arena) Free() {
	if a.freed {
		panic("arena: cannot free an already freed arena")
	}
	walk := a.current
	for walk != nil {
		next := walk.next
		walk.buf = nil
		walk.next = nil
		walk = next
	}
	a.current = nil
	a.offset = 0
	a.base = nil
	a.end = 0
	a.freed = true
}

// Mark saves the current allocation position. Pair with Release to bound
// the lifetime of everything allocated between the two calls:
//
//	mark := scratch.Mark()
//	frame := Alloc[CallFrame](scratch)
//	frame.locals = AllocSlice[Value](scratch, fn.LocalCount)
//	scratch.Release(mark) // frame and locals are gone in O(1)
func (a *Arena) Mark() Mark {
	return Mark{base: a.base, offset: a.offset, chunk: a.current}
}

// Release rewinds the arena to the position saved by Mark. Chunks allocated
// after the mark are abandoned and collected by the GC. Does not zero memory.
func (a *Arena) Release(m Mark) {
	a.current = m.chunk
	a.offset = m.offset
	a.base = m.base
	a.end = uintptr(len(m.chunk.buf))
}

// Stats walks the chunk chain and returns the total chunk count and byte capacity.
// Useful for tuning initial arena sizes — call it after a representative workload.
func (a *Arena) Stats() ArenaStats {
	totalChunks := 0
	totalBytes := 0
	walk := a.current
	for walk != nil {
		totalChunks++
		totalBytes += len(walk.buf)
		walk = walk.next
	}
	return ArenaStats{totalChunks, totalBytes}
}

func (a *Arena) Remaining() int { return int(a.end) - int(a.offset) }
func (a *Arena) Used() int      { return int(a.offset) }

// Alloc allocates one T in the arena and returns a pointer to it.
// Memory is zero-initialised on first use (new chunks) and after Reset.
func Alloc[T any](a *Arena) *T {
	var t T
	ptr := a.alloc(unsafe.Sizeof(t), unsafe.Alignof(t))
	return (*T)(ptr)
}

// Returns nil for negative count; panics on integer overflow.
// AllocSlice allocates a slice of count T's with len == cap == count.
func AllocSlice[T any](a *Arena, count int) []T {
	if count < 0 {
		return nil
	}
	var t T
	elemSize := unsafe.Sizeof(t)
	if count > 0 && uintptr(count) >= ^uintptr(0)/elemSize {
		panic("arena: overflow")
	}
	ptr := a.alloc(elemSize*uintptr(count), unsafe.Alignof(t))
	return unsafe.Slice((*T)(ptr), count)
}

// AllocSliceCap allocates capacity T's but returns a slice with len=length.
// Use this when building arrays whose final size is unknown — standard append
// can grow within the arena-owned capacity before spilling to the heap:
//
//	instrs := AllocSliceCap[Instr](permanent, 0, 64)
//	instrs  = append(instrs, Instr{Op: OpLoad, A: 0})
func AllocSliceCap[T any](a *Arena, length, capacity int) []T {
	return AllocSlice[T](a, capacity)[:length]
}

// AllocString copies s into the arena and returns a string backed by it.
// Use the permanent arena to intern identifiers and literals so they outlive
// the scratch arena:
//
//	fn.Name = AllocString(permanent, "main")
func AllocString(a *Arena, s string) string {
	if s == "" {
		return ""
	}
	buf := AllocSlice[byte](a, len(s))
	copy(buf, s)
	return unsafe.String(&buf[0], len(s))
}

// AllocBytes copies b into the arena and returns the arena-backed slice.
func AllocBytes(a *Arena, b []byte) []byte {
	if b == nil {
		return nil
	}
	buf := AllocSlice[byte](a, len(b))
	copy(buf, b)
	return buf
}

// AppendToArena used to for realloc so that if the initial capactiy is reached
// it doesn't fall back to the heap, instead we grow the same slice with 2x capcity
func AppendToArena[T any](a *Arena, s []T, v T) []T {
	if len(s) < cap(s) {
		return append(s, v)
	}
	grown := AllocSliceCap[T](a, len(s), cap(s)*2)
	copy(grown, s)
	return append(grown, v)
}

// Copy writes v into the arena and returns a pointer to the arena-owned copy.
// Use this to pin a stack value into a longer lifetime:
//
//	node := Copy(permanent, ASTNode{Kind: KindAdd, Left: l, Right: r})
func Copy[T any](a *Arena, v T) *T {
	ptr := Alloc[T](a)
	*ptr = v
	return ptr
}

// align rounds offset up to the next multiple of alignment (must be a power of 2).
// mask = alignment-1 sets the low bits; adding it to offset guarantees an overflow
// into the next aligned address, then &^ mask clears those low bits.
func align(offset, alignment uintptr) uintptr {
	mask := alignment - 1
	return (offset + mask) &^ mask
}
