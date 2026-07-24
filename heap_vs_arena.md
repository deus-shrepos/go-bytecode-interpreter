# Heap Allocation, Arena Allocation, and Escape Analysis in Go

## 1. The Two Worlds of Memory

Every running program operates with two kinds of memory: memory whose lifetime is known at compile time, and memory whose lifetime is not. The first kind lives on the stack. The second lives on the heap.

The stack is simple. When you call a function, the runtime bumps a pointer forward by the size of the function's frame. When the function returns, it bumps the pointer back. Zero bookkeeping. Zero contention. Extremely fast. The constraint is that nothing allocated on the stack can outlive the function that created it.

The heap handles everything else. When you need memory that will outlive its allocation site, or whose size is not known until runtime, or that needs to be shared across goroutines, it goes on the heap. The tradeoff is cost. The allocator must track which regions are in use and which are free. Eventually, memory that is no longer reachable must be reclaimed. In Go, this reclamation is handled by a concurrent tri-color mark-and-sweep garbage collector.

Understanding the tension between stack and heap, and what forces a value from one to the other, is essential to writing allocation-efficient Go.

---

## 2. Go's Garbage Collector and Why Allocations Are Expensive

Go's GC is generational in spirit but non-generational in implementation. It runs concurrently with the program, scanning the heap to find which objects are still reachable from roots (globals, goroutine stacks, and the finalizer queue). Unreachable objects are reclaimed.

The cost of the GC is not just pause time. The more objects on the heap, the more work the GC must do on every cycle. Each allocation:

1. Acquires memory from a size-class span (Go's allocator, `mcache`, organizes the heap into size classes ranging from 8 bytes to 32KB).
2. Potentially triggers a GC cycle if the heap has grown past the trigger threshold.
3. Contributes to scan work — the GC must eventually examine every live pointer on the heap.

For latency-sensitive programs such as virtual machines, parsers, or game engines, even a well-tuned GC imposes unpredictable pauses and CPU overhead. The goal is not to avoid heap allocation entirely, but to understand precisely when it occurs and whether it is necessary.

---

## 3. Escape Analysis: What It Is and What It Is Not

Go determines whether a value lives on the stack or the heap through a compiler pass called escape analysis. The analysis asks one question for every value in a function: can this value be proven to not outlive the current stack frame? If yes, it stays on the stack. If the compiler cannot prove this, the value escapes to the heap.

You can inspect escape analysis decisions by building with:

```
go build -gcflags="-m" ./...
```

Or for more detailed output:

```
go build -gcflags="-m=2" ./...
```

The critical thing to understand about this output is what it represents: a conservative static approximation of what might happen at runtime. The compiler must be sound — it cannot allow a stack allocation where a heap allocation is required. But it is not required to be precise. It will flag things as escaping even when, at runtime, no heap allocation actually occurs.

This distinction is the source of enormous confusion.

---

## 4. Reading Escape Analysis Output

### 4.1 What the flags mean

```
-m    print optimization decisions (inlining and escape analysis)
-m=2  print the full escape flow graph
```

With `-m=2`, each escape decision includes a flow trace showing exactly why the compiler decided a value must live on the heap. For example:

```
./main.go:12:2: moved to heap: x
./main.go:12:2:   flow: {heap} <- &x:
./main.go:12:2:     from &x (address-of) at ./main.go:13:10
./main.go:12:2:     from store(p, &x) (call parameter) at ./main.go:13:5
```

This says: `x` was moved to the heap because its address was taken and passed to a function that stores it somewhere the compiler cannot track.

### 4.2 Common triggers for heap escape

**Returning a pointer to a local variable:**

```go
func newFoo() *Foo {
    f := Foo{x: 1}
    return &f  // f escapes: its address outlives the function
}
```

Here the compiler correctly identifies that `f` cannot live on `newFoo`'s stack frame because the caller receives a pointer to it. `f` is moved to the heap.

**Storing a pointer in a data structure that escapes:**

```go
type Node struct {
    val  int
    next *Node
}

func build() *Node {
    a := Node{val: 1}
    b := Node{val: 2, next: &a}  // &a stored in b
    return &b                     // b escapes, and a escapes because b holds &a
}
```

Both `a` and `b` escape because the compiler traces pointer flows and determines that `a`'s address is reachable from the return value.

**Interface boxing:**

```go
func print(v interface{}) {
    fmt.Println(v)
}

func main() {
    x := 3.14
    print(x)  // x escapes: assigned to interface{}
}
```

When a concrete value is assigned to an `interface{}` (or any interface), Go must store the value somewhere that the interface header can point to. For values larger than a pointer, this means a heap allocation. Even `fmt.Printf("%g", myFloat64)` allocates because the `...interface{}` parameter forces boxing.

**Slices and maps with dynamic sizes:**

```go
func makeSlice(n int) []int {
    return make([]int, n)  // n unknown at compile time, escapes to heap
}
```

If the size of a slice is not a compile-time constant, or if it exceeds a certain threshold (currently 64KB), the backing array goes to the heap.

**Closures capturing variables:**

```go
func counter() func() int {
    n := 0
    return func() int {
        n++    // n is captured by reference; it escapes
        return n
    }
}
```

`n` must outlive `counter`'s frame because the returned closure holds a reference to it.

### 4.3 The important distinction: static vs. dynamic

The escape analysis output describes what the compiler decided, not what happens at runtime. Consider:

```go
func process(items []Item) {
    buf := make([]byte, 0, 64)
    for _, item := range items {
        buf = append(buf, item.data...)
    }
    write(buf)
}
```

If `write` is not inlined and the compiler cannot prove `buf` does not escape through `write`, it may report `buf` as escaping. But if `write` only reads the slice and does not store it, no heap allocation actually occurs at runtime. The static analysis is conservative.

The benchmark is the ground truth:

```go
func BenchmarkProcess(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        process(testItems)
    }
}
```

`b.ReportAllocs()` reports `allocs/op` — actual calls to `runtime.mallocgc`. If it shows 0, no heap allocation occurred regardless of what `-m` said.

---

## 5. What `append` Looks Like to the Escape Analyzer

`append` is a built-in function with special semantics. When `len(s) < cap(s)`, it stores the new element in-place and returns a new slice header (same pointer, incremented length, same capacity). When `len(s) == cap(s)`, it allocates a new backing array — typically 2x the old capacity — copies the old data, appends the new element, and returns a header pointing at the new array.

The escape analyzer sees `append` and reasons as follows:

1. The backing array might need to be reallocated.
2. If it is reallocated, a new allocation occurs on the heap.
3. Therefore, the storage for the result of `append` may live on the heap.

It flags this regardless of whether the slice has room or not, because it cannot prove at compile time that `len < cap` will always hold. This is a conservative but sound decision.

Consider the difference between these two programs:

```go
// Program A: backing array definitely escapes
func escapes() []int {
    s := make([]int, 0)   // no capacity hint, starts at 0
    for i := 0; i < 100; i++ {
        s = append(s, i)  // reallocates multiple times: 0->1->2->4->8->...
    }
    return s
}
```

```go
// Program B: no reallocation at runtime, but escape analyzer warns anyway
func noRealloc() []int {
    s := make([]int, 0, 100)  // pre-sized
    for i := 0; i < 100; i++ {
        s = append(s, i)      // always len < cap, no reallocation
    }
    return s
}
```

In Program B, the backing array is allocated exactly once (by `make`) and `append` never calls the allocator. But the escape analyzer will still report the append as potentially escaping, because it performs a syntactic analysis: it sees `append`, it sees that the result could be a new pointer, it cannot rule out reallocation.

The escape warning for `append` in Program B is a false positive. The runtime counter (`allocs/op`) will show 1 alloc for the `make`, not one per `append`.

---

## 6. Arena Allocation: The Idea

A heap allocator must solve the general problem: arbitrary objects of arbitrary sizes with arbitrary lifetimes, allocated and freed in arbitrary order. This generality is what makes it expensive.

An arena allocator abandons generality to gain speed. It makes a single large allocation upfront, then satisfies individual requests by advancing a pointer through that region. Individual objects are never freed. Instead, the entire arena is freed at once when the work it represents is complete.

```
Arena buffer:
[  obj1  ][  obj2  ][  obj3  ][ free .............. ]
                               ^
                               offset (bump pointer)
```

Allocation is O(1) and branchless in the common case: align the offset, add the requested size, return the old offset as the pointer. No free list traversal. No size-class lookup. No GC scan work for objects inside the arena (they are all within the single root allocation).

Arenas are appropriate when a group of allocations shares a lifetime. A bytecode interpreter is a canonical example: the chunk (bytecode), the constant pool, and the value stack all live for the duration of a single interpretation run. Allocating them in an arena and freeing the entire arena at the end costs exactly one GC allocation (the arena's backing buffer) regardless of how many objects were placed inside it.

---

## 7. Arenas in Go: Implementation and Caveats

Go does not have native arena support in stable releases. Arenas must be emulated using `unsafe`. The standard approach is to allocate a large `[]byte` from the GC heap and carve objects out of it using `unsafe.Pointer` arithmetic.

```go
type Arena struct {
    buf    []byte
    offset int
}

func NewArena(size int) *Arena {
    return &Arena{buf: make([]byte, size)}
}

func Alloc[T any](a *Arena) *T {
    var zero T
    size := int(unsafe.Sizeof(zero))
    align := int(unsafe.Alignof(zero))
    // round up offset to alignment
    a.offset = (a.offset + align - 1) &^ (align - 1)
    ptr := unsafe.Pointer(&a.buf[a.offset])
    a.offset += size
    return (*T)(ptr)
}
```

The critical observation is that from the Go compiler's perspective, the backing `[]byte` is a normal GC-heap allocation. The `unsafe.Pointer` arithmetic that carves objects from it is opaque to both the escape analyzer and the GC. The GC sees one large `[]byte` and tracks it as a single object. The individual `*T` pointers returned by `Alloc` point into the interior of that `[]byte`.

This creates an important consequence: the GC does not scan inside the arena for pointers unless you tell it to. If any arena-allocated struct contains a Go pointer (a `*T` or a slice header containing a non-arena pointer), the GC will not find it through the arena. This can cause pointers to be collected while still in use. For structs containing only scalar values (integers, floats, booleans), this is not a concern.

### 7.1 Pre-sized slices in an arena

A common pattern is to allocate a slice's backing array from the arena and use a known capacity:

```go
func AllocSliceCap[T any](a *Arena, length, capacity int) []T {
    backing := AllocSlice[T](a, capacity)
    return backing[:length]
}
```

This returns a slice with `cap = capacity` and `len = length`, backed by arena memory. When you later call `append` on this slice, as long as `len < cap`, the runtime writes directly into the arena's buffer. No GC allocation occurs.

The slice header itself (a 24-byte struct: pointer, len, cap) lives wherever you store it — typically in a struct that may be on the GC heap. Writing an updated slice header back to that struct is a write to an existing heap address. It is not a new allocation.

---

## 8. Why Escape Analysis Warns About Arena-Backed Slices

This is where static analysis and runtime behavior diverge most sharply, and where careful interpretation matters.

Consider a struct allocated from an arena:

```go
type Chunk struct {
    Code []OpCode
}

chunk := ArenaAlloc[Chunk](arena)
chunk.Code = ArenaSliceCap[OpCode](arena, 0, 64)
```

Now you call a method that appends to `chunk.Code`:

```go
func (c *Chunk) WriteChunk(op OpCode) {
    c.Code = append(c.Code, op)
}
```

The escape analyzer will report something like:

```
append(c.Code, op) escapes to heap
  flow: {heap} <- &{storage for append(c.Code, op)}:
    from append(c.Code, op) (spill) at chunks.go:27
    from c.Code = append(c.Code, op) (assign) at chunks.go:27
```

The flow trace is saying: the result of `append` (a new slice header) is assigned to `c.Code`, which is a field of `c`, and `c` is a pointer that escapes to the heap. Therefore, the storage for the append result is at a heap address.

This warning is technically correct but practically irrelevant. Read it carefully: it says the *result* (the updated slice header) lives at a heap address. That heap address is `c.Code` itself, which already exists. No new allocation occurs. The compiler is noting that it cannot keep the intermediate slice header value in a register and must spill it to a memory location that happens to be on the heap.

At runtime, when `len(c.Code) < cap(c.Code)`:

1. The runtime checks `len` vs `cap` in the slice header.
2. It writes the new element to `c.Code.ptr[c.Code.len]`.
3. It returns a new slice header with `len + 1`.
4. The caller stores this header back into `c.Code`.

Steps 2 and 4 write to the arena's backing buffer. No call to `runtime.mallocgc`. No GC allocation.

The only way the escape analyzer's warning corresponds to an actual allocation is if `append` overflows capacity and the runtime must call `growslice`. This is what you must prevent through careful pre-sizing.

### 8.1 The cascade effect

One escape can produce dozens of warnings that all trace back to a single root cause. In the bytecode interpreter example, passing `&chunk` (a pointer to a local variable) to `NewVM` caused `chunk` to escape to the heap. Every subsequent `append` call on `chunk.Code` and `chunk.Lines` then generated its own warning, all cascading from that one root escape.

```
chunk escapes to heap           <- ROOT CAUSE
  from &chunk (address-of)
  from NewVM(arena, &chunk)

append(c.Code, op) escapes      <- CASCADE
append(c.Lines, l) escapes      <- CASCADE
append(c.Lines, l) escapes      <- CASCADE (second occurrence same line)
```

The fix was to allocate `chunk` from the arena itself, so no stack-to-heap escape occurred. After that fix, the cascade warnings disappeared because the pointer passed to `NewVM` was already an arena pointer, not a pointer to a stack variable. The `append` warnings for writing into `chunk.Code` persisted at the static level — because the arena's buffer is still GC-heap memory — but produced zero actual allocations.

---

## 9. Measuring What Actually Happens

Static analysis tells you what could happen. Benchmarks tell you what does happen.

### 9.1 b.ReportAllocs

```go
func BenchmarkVM(b *testing.B) {
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        arena := NewArena(1 << 18)
        chunk := ArenaAlloc[Chunk](arena)
        *chunk = NewChunk(arena)
        chunk.WriteChunk(OP_CONST)
        chunk.WriteChunk(OP_RETURN)
        v := NewVM(arena, chunk)
        v.Run()
        arena.Free()
    }
}
```

Output:

```
BenchmarkVM-16    68456    18593 ns/op    262225 B/op    3 allocs/op
```

Three allocations. Not dozens, as the escape analysis warnings might suggest. The three correspond to:

1. The arena's backing `[]byte` — `make([]byte, 262144)`. This is the arena itself. It is unavoidable and intentional.
2. The arena's internal bookkeeping struct — a small node in the arena's linked list of chunks.
3. The VM struct — returned as `*VM` from `NewVM`, which forces it to the GC heap.

Every `append` inside `WriteChunk` and every `Push` inside `Run` contributed zero allocations. The pre-sized slices stayed within their arena-backed capacity.

### 9.2 runtime.ReadMemStats

For production code where you cannot introduce benchmark overhead, `runtime.ReadMemStats` provides cumulative allocation counters:

```go
var before, after runtime.MemStats
runtime.ReadMemStats(&before)
doWork()
runtime.ReadMemStats(&after)
fmt.Printf("allocs: %d, bytes: %d\n",
    after.Mallocs - before.Mallocs,
    after.TotalAlloc - before.TotalAlloc)
```

`Mallocs` counts calls to the allocator. `TotalAlloc` counts bytes allocated (not bytes currently live). Both are cumulative since program start, so you take a delta.

### 9.3 pprof heap profiling

For larger programs where you need to find which allocation sites dominate:

```go
import _ "net/http/pprof"

// in main:
go http.ListenAndServe(":6060", nil)
```

Then:

```
go tool pprof http://localhost:6060/debug/pprof/heap
```

This gives a full call graph of allocation sites weighted by bytes. The `-m` output tells you what might allocate. The heap profile tells you what actually allocates and how much.

---

## 10. The Rules for Correct Interpretation

After working through the mechanics, a clear set of rules emerges for interpreting escape analysis output in the presence of arena-allocated memory.

**Rule 1: Find the root escape, ignore the cascade.**
One value escaping to the heap causes every value reachable through it to also be flagged. Identify the root cause — usually a pointer passed to a function that stores it — and fix that. The cascade warnings will disappear or become harmless.

**Rule 2: An `append` warning is not necessarily an `append` allocation.**
The escape analyzer warns when the result of `append` is stored at a heap address. This includes updating a slice header field inside an existing heap-allocated struct. When `len < cap`, the backing array does not move. The warning is accurate about where the slice header lives, not about whether a new backing array was allocated.

**Rule 3: Arena-backed memory is still GC-heap memory to the escape analyzer.**
The compiler does not know about your arena. A `*T` returned from `unsafe.Pointer` arithmetic inside a `[]byte` looks like a heap pointer. Writes through that pointer will be flagged as writes to the heap. This cannot be avoided through source-level changes — it is inherent to how Go implements arenas with `unsafe`.

**Rule 4: Interface boxing always allocates.**
Any concrete value assigned to `interface{}` — including all calls to `fmt.Printf`, `fmt.Println`, and similar variadic functions — causes a heap allocation if the value is larger than a pointer. This is separate from arena allocation. If you call `fmt.Printf("%g\n", myFloat64)` inside your hot loop, you will see one allocation per call regardless of how well-managed your arena is. This is often the dominant source of allocations in programs that use arenas for data structures but retain `fmt` for output.

**Rule 5: Benchmark with `b.ReportAllocs()`. Static analysis is advisory.**
The escape analyzer is a conservative tool. It cannot give you false negatives (it will not miss a real heap escape) but it can give you false positives (it will warn about things that do not actually allocate). The benchmark is the only authoritative source for runtime allocation counts. Use `-m=2` to understand the structure of escapes and find root causes; use `b.ReportAllocs()` to confirm whether they matter.

---

## 11. Arena Limitations in Go

Arenas in Go are powerful but carry constraints that do not exist in languages with manual memory management.

**Pointer lifetimes must be respected manually.** The GC does not scan inside the arena for pointers to other GC-managed objects. If an arena-allocated struct holds a `*T` where `T` is on the GC heap, the GC may collect `T` while the arena struct still holds the pointer. You must ensure arena-allocated structs contain only other arena pointers, only scalars, or that you maintain an explicit reference to any GC objects they point to.

**The arena's backing buffer is a GC root.** The GC will not collect the arena's `[]byte` while it is live. Everything inside it is implicitly kept alive. This means arena memory cannot be partially reclaimed — you must free the entire arena at once. A Mark/Release pattern (saving the current offset and rewinding to it later) can scope sub-allocations within an arena, but the backing buffer itself is not freed until the arena is explicitly released.

**`append` will fall back to the GC heap if capacity is exceeded.** If you `append` past the pre-allocated capacity of an arena-backed slice, Go's runtime calls `growslice`, which allocates a new backing array from the GC heap. The old arena memory is not freed (the arena has no per-object free), and you now have a slice backed by GC heap memory. Pre-sizing is not optional — it is a correctness requirement for keeping allocations within the arena.

**Write barriers apply.** When you write a GC-managed pointer into a location the GC can reach (including fields of structs inside an arena-backed `[]byte`), the GC's write barrier fires. Write barriers add overhead to pointer writes. For structs containing only scalars, this is a non-issue. For structs with pointer fields, the barrier cost is worth measuring.

---

## 12. Summary

The gap between escape analysis warnings and actual runtime allocations is widest when using arena allocation. The escape analyzer sees your arena's backing buffer as a heap allocation and flags every pointer write into it as a heap write. This is technically correct but operationally misleading: writing a slice header into an existing heap struct is not the same as allocating a new struct.

The practical workflow is:

1. Use `-gcflags="-m=2"` to understand the escape structure and find root causes.
2. Fix genuine root escapes — values that should live in the arena but are being moved to the GC heap because of how pointers are passed.
3. Accept that arena-internal operations will show escape warnings that do not correspond to real allocations.
4. Verify with `b.ReportAllocs()` that the actual allocation count matches your model.
5. Watch for `fmt` calls, interface boxing, and `append` overflows as the real sources of unintended allocations.

The goal is not a clean `-m` output. The goal is a low and predictable allocation count at runtime, confirmed by measurement.
