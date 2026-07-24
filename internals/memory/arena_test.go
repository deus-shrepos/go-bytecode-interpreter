package memory

import (
	"testing"
	"unsafe"
)

// --- alignment ---

func TestAllocAlignment(t *testing.T) {
	a := NewArena(256)

	check := func(ptr unsafe.Pointer, alignment uintptr, label string) {
		t.Helper()
		if uintptr(ptr)%alignment != 0 {
			t.Errorf("%s: pointer %x not aligned to %d", label, uintptr(ptr), alignment)
		}
	}

	_ = Alloc[int8](a)
	check(unsafe.Pointer(Alloc[int16](a)), 2, "int16 after int8")

	_ = Alloc[int8](a)
	check(unsafe.Pointer(Alloc[int32](a)), 4, "int32 after int8")

	_ = Alloc[int8](a)
	check(unsafe.Pointer(Alloc[int64](a)), 8, "int64 after int8")
}

// --- value correctness ---

func TestAllocValue(t *testing.T) {
	a := NewArena(64)
	p := Alloc[int64](a)
	*p = 0xdeadbeef
	if *p != 0xdeadbeef {
		t.Fatalf("want 0xdeadbeef, got %x", *p)
	}
}

func TestAllocSliceValues(t *testing.T) {
	a := NewArena(64)
	s := AllocSlice[int32](a, 8)

	if len(s) != 8 {
		t.Fatalf("len: want 8, got %d", len(s))
	}
	for i := range s {
		s[i] = int32(i * 10)
	}
	for i, v := range s {
		if v != int32(i*10) {
			t.Errorf("s[%d]: want %d, got %d", i, i*10, v)
		}
	}
}

// --- offset accounting ---

func TestOffsetAccounting(t *testing.T) {
	a := NewArena(64)

	AllocSlice[int8](a, 4) // 4 bytes, align 1 -> offset = 4
	if a.Used() != 4 {
		t.Errorf("after int8x4: Used() = %d, want 4", a.Used())
	}

	AllocSlice[int16](a, 2) // align 2, offset already 4 -> offset = 8
	if a.Used() != 8 {
		t.Errorf("after int16x2: Used() = %d, want 8", a.Used())
	}

	Alloc[int64](a) // align 8, offset already 8 -> offset = 16
	if a.Used() != 16 {
		t.Errorf("after int64: Used() = %d, want 16", a.Used())
	}

	if a.Remaining() != 64-16 {
		t.Errorf("Remaining() = %d, want %d", a.Remaining(), 64-16)
	}
}

// --- grow ---

func TestGrowBasic(t *testing.T) {
	a := NewArena(8)

	p1 := Alloc[int64](a)
	*p1 = 111

	p2 := Alloc[int64](a)
	*p2 = 222

	if *p1 != 111 {
		t.Errorf("p1 after grow: want 111, got %d", *p1)
	}
	if *p2 != 222 {
		t.Errorf("p2 after grow: want 222, got %d", *p2)
	}
	if a.Remaining() <= 0 {
		t.Errorf("Remaining() after grow: want > 0, got %d", a.Remaining())
	}
}

func TestGrowChain(t *testing.T) {
	a := NewArena(8)

	ptrs := make([]*int64, 10)
	for i := range ptrs {
		p := Alloc[int64](a)
		*p = int64(i * 100)
		ptrs[i] = p
	}
	for i, p := range ptrs {
		if *p != int64(i*100) {
			t.Errorf("ptrs[%d]: want %d, got %d", i, i*100, *p)
		}
	}
}

func TestGrowAlignment(t *testing.T) {
	a := NewArena(8)

	_ = Alloc[int8](a)
	_ = Alloc[int64](a) // triggers grow

	p := Alloc[int64](a)
	if uintptr(unsafe.Pointer(p))%8 != 0 {
		t.Errorf("int64 in grown chunk not aligned: %x", uintptr(unsafe.Pointer(p)))
	}
}

// --- pointer stability ---

func TestPointerStability(t *testing.T) {
	a := NewArena(128)

	p1 := Alloc[int64](a)
	*p1 = 0xAAAA
	p2 := Alloc[int64](a)
	*p2 = 0xBBBB
	s := AllocSlice[int32](a, 4)
	for i := range s {
		s[i] = int32(i)
	}

	_ = AllocSlice[int8](a, 32)

	if *p1 != 0xAAAA {
		t.Errorf("p1 clobbered: got %x", *p1)
	}
	if *p2 != 0xBBBB {
		t.Errorf("p2 clobbered: got %x", *p2)
	}
	for i, v := range s {
		if v != int32(i) {
			t.Errorf("s[%d] clobbered: got %d", i, v)
		}
	}
}

// --- reset ---

func TestReset(t *testing.T) {
	a := NewArena(64)

	_ = AllocSlice[int64](a, 4)
	a.Reset()

	if a.Used() != 0 {
		t.Errorf("Used() after Reset: want 0, got %d", a.Used())
	}
	if a.Remaining() != 64 {
		t.Errorf("Remaining() after Reset: want 64, got %d", a.Remaining())
	}

	p := Alloc[int64](a)
	*p = 42
	if *p != 42 {
		t.Errorf("alloc after Reset: want 42, got %d", *p)
	}
}

func TestResetAfterGrow(t *testing.T) {
	a := NewArena(8)
	_ = Alloc[int64](a)
	_ = Alloc[int64](a)

	a.Reset()

	if a.Used() != 0 {
		t.Errorf("Used() after Reset: want 0, got %d", a.Used())
	}
	if a.base != unsafe.Pointer(&a.current.buf[0]) {
		t.Error("base not restored after Reset")
	}
	if a.end != uintptr(len(a.current.buf)) {
		t.Errorf("end not restored: want %d, got %d", len(a.current.buf), a.end)
	}

	p := Alloc[int64](a)
	*p = 99
	if *p != 99 {
		t.Errorf("alloc after Reset+grow: want 99, got %d", *p)
	}
}

func TestResetDropsChain(t *testing.T) {
	a := NewArena(8)
	_ = Alloc[int64](a)
	_ = Alloc[int64](a)

	a.Reset()

	if a.current.next != nil {
		t.Error("Reset did not drop the chunk chain")
	}
}

func TestResetZeroesMemory(t *testing.T) {
	a := NewArena(128)
	s := AllocSlice[byte](a, 64)
	for i := range s {
		s[i] = 0xFF
	}

	a.Reset()

	// The used region must be cleared; new allocations in the same region get zeroes.
	s2 := AllocSlice[byte](a, 64)
	for i, v := range s2 {
		if v != 0 {
			t.Errorf("byte[%d] after Reset: want 0, got %x", i, v)
		}
	}
}

// --- mark / release ---

func TestMarkReleaseBasic(t *testing.T) {
	a := NewArena(128)

	p1 := Alloc[int64](a)
	*p1 = 0xCAFE
	mark := a.Mark()
	beforeUsed := a.Used()

	p2 := Alloc[int64](a)
	*p2 = 0xBEEF

	a.Release(mark)

	if a.Used() != beforeUsed {
		t.Errorf("Used() after Release: want %d, got %d", beforeUsed, a.Used())
	}
	if *p1 != 0xCAFE {
		t.Errorf("pre-mark value clobbered: got %x", *p1)
	}
}

func TestMarkReleaseAcrossGrow(t *testing.T) {
	// Take a mark before the arena grows and verify Release rewinds
	// back to the original chunk, not the grown one.
	a := NewArena(32)

	_ = AllocSlice[int32](a, 4) // 16 bytes, offset = 16
	mark := a.Mark()

	_ = AllocSlice[int64](a, 8) // 64 bytes, forces grow

	a.Release(mark)

	if a.current != mark.chunk {
		t.Error("Release: current chunk not restored to pre-grow chunk")
	}
	if a.offset != mark.offset {
		t.Errorf("Release: offset not restored: want %d, got %d", mark.offset, a.offset)
	}
	if a.base != mark.base {
		t.Error("Release: base not restored")
	}
	if a.end != uintptr(len(mark.chunk.buf)) {
		t.Errorf("Release: end not restored: want %d, got %d", len(mark.chunk.buf), a.end)
	}
}

func TestMarkReleaseNested(t *testing.T) {
	// Simulate a 4-level interpreter call stack. Each frame allocates a return
	// value and a variable-length locals slice. Frames are released in LIFO order.
	a := NewArena(1024)

	type frame struct {
		retVal int64
		locals []int32
	}

	const depth = 4
	var marks [depth]Mark
	var frames [depth]*frame

	for i := range depth {
		marks[i] = a.Mark()
		f := Alloc[frame](a)
		f.retVal = int64(i * 100)
		f.locals = AllocSlice[int32](a, i+2)
		for j := range f.locals {
			f.locals[j] = int32(i*10 + j)
		}
		frames[i] = f
	}

	for i, f := range frames {
		if f.retVal != int64(i*100) {
			t.Errorf("frame[%d].retVal: want %d, got %d", i, i*100, f.retVal)
		}
		for j, v := range f.locals {
			if want := int32(i*10 + j); v != want {
				t.Errorf("frame[%d].locals[%d]: want %d, got %d", i, j, want, v)
			}
		}
	}

	for i := depth - 1; i >= 0; i-- {
		a.Release(marks[i])
		if a.offset != marks[i].offset {
			t.Errorf("after Release(%d): offset = %d, want %d", i, a.offset, marks[i].offset)
		}
	}

	if a.offset != 0 {
		t.Errorf("after all releases: offset = %d, want 0", a.offset)
	}
}

// --- alloc string ---

func TestAllocStringEmpty(t *testing.T) {
	a := NewArena(64)
	s := AllocString(a, "")
	if s != "" {
		t.Errorf("want \"\", got %q", s)
	}
	if a.Used() != 0 {
		t.Errorf("empty string must not consume arena bytes, Used() = %d", a.Used())
	}
}

func TestAllocStringContent(t *testing.T) {
	a := NewArena(64)
	s := AllocString(a, "hello")
	if s != "hello" {
		t.Errorf("want \"hello\", got %q", s)
	}
	if a.Used() != 5 {
		t.Errorf("Used() after 5-byte string: want 5, got %d", a.Used())
	}
}

func TestAllocStringIndependence(t *testing.T) {
	a := NewArena(64)
	src := "hello"
	got := AllocString(a, src)

	if got != "hello" {
		t.Fatalf("content wrong: got %q", got)
	}
	// The arena copy must be a distinct allocation, not a reference to src's data.
	if unsafe.StringData(src) == unsafe.StringData(got) {
		t.Error("AllocString returned the source string's backing data, not a copy")
	}
}

// --- alloc bytes ---

func TestAllocBytesNil(t *testing.T) {
	a := NewArena(64)
	b := AllocBytes(a, nil)
	if b != nil {
		t.Errorf("AllocBytes(nil): want nil, got %v", b)
	}
}

func TestAllocBytesContent(t *testing.T) {
	a := NewArena(64)
	src := []byte{1, 2, 3, 4}
	got := AllocBytes(a, src)

	if len(got) != len(src) {
		t.Fatalf("len: want %d, got %d", len(src), len(got))
	}
	for i, v := range got {
		if v != src[i] {
			t.Errorf("got[%d]: want %d, got %d", i, src[i], v)
		}
	}
}

func TestAllocBytesIndependence(t *testing.T) {
	a := NewArena(64)
	src := []byte{10, 20, 30}
	got := AllocBytes(a, src)

	src[0] = 0xFF
	if got[0] != 10 {
		t.Errorf("arena copy affected by source mutation: want 10, got %d", got[0])
	}
}

// --- copy ---

func TestCopy(t *testing.T) {
	type point struct{ x, y int64 }

	a := NewArena(64)
	p := point{x: 10, y: 20}
	ap := Copy(a, p)

	if ap.x != 10 || ap.y != 20 {
		t.Errorf("Copy: want {10 20}, got {%d %d}", ap.x, ap.y)
	}

	// Mutating the source must not affect the arena copy.
	p.x = 999
	if ap.x != 10 {
		t.Errorf("arena copy affected by source mutation: got %d", ap.x)
	}
}

// --- stats ---

func TestStats(t *testing.T) {
	a := NewArena(16)

	s := a.Stats()
	if s.TotalChunks != 1 || s.TotalBytes != 16 {
		t.Errorf("initial: want {1, 16}, got {%d, %d}", s.TotalChunks, s.TotalBytes)
	}

	_ = AllocSlice[int64](a, 8) // 64 bytes into a 16-byte arena — forces a grow

	s = a.Stats()
	if s.TotalChunks < 2 {
		t.Errorf("after grow: want >= 2 chunks, got %d", s.TotalChunks)
	}
	if s.TotalBytes <= 16 {
		t.Errorf("after grow: TotalBytes = %d, want > 16", s.TotalBytes)
	}
}

// --- AllocSlice guards ---

func TestAllocSliceNegativeCount(t *testing.T) {
	a := NewArena(64)
	s := AllocSlice[int64](a, -1)
	if s != nil {
		t.Errorf("negative count: want nil, got non-nil slice of len %d", len(s))
	}
}

func TestAllocSliceOverflow(t *testing.T) {
	a := NewArena(64)
	defer expectPanic(t, "overflow")
	AllocSlice[int64](a, int(^uintptr(0)/8))
}

// --- free ---

func TestFreeDoublePanic(t *testing.T) {
	a := NewArena(64)
	a.Free()
	defer expectPanic(t, "double Free")
	a.Free()
}

func TestAllocAfterFreePanic(t *testing.T) {
	a := NewArena(64)
	a.Free()
	defer expectPanic(t, "Alloc after Free")
	Alloc[int64](a)
}

func TestResetAfterFreePanic(t *testing.T) {
	a := NewArena(64)
	a.Free()
	defer expectPanic(t, "Reset after Free")
	a.Reset()
}

func expectPanic(t *testing.T, label string) {
	t.Helper()
	if r := recover(); r == nil {
		t.Errorf("%s: expected panic, got none", label)
	}
}
