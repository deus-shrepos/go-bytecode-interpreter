package vm

import (
	c "go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"
)

// memSnapshot is one labeled observation of the three memory domains:
// the Go runtime's heap/GC state, the arena, and the VM value stack.
type memSnapshot struct {
	label string

	// Go runtime (from runtime.ReadMemStats)
	heapLive   uint64 // HeapAlloc: live heap bytes at this instant
	totalAlloc uint64 // TotalAlloc: cumulative; diff between rows = alloc work in window
	mallocs    uint64 // Mallocs: cumulative heap objects allocated; diff = mallocgc calls in window
	numGC      uint32 // cumulative GC cycles; diff = cycles fired in window
	lastPause  uint64 // PauseNs of the most recent cycle (only meaningful if numGC moved)
	stackInuse uint64 // StackInuse: Go-managed goroutine stack memory

	// arena
	arenaBytes  int
	arenaChunks int
	arenaUsed   int // current-chunk used only; old chunks' usage is unrecorded

	// VM value stack
	vmStackLen int
	vmStackCap int
}

// memProbe records snapshots at event boundaries and prints a delta table.
// The snapshots slice is preallocated so Snapshot itself never allocates —
// otherwise the probe would pollute its own TotalAlloc measurements.
type memProbe struct {
	snaps []memSnapshot
	vm    *VM
	arena *memory.Arena
}

func newMemProbe(vm *VM, arena *memory.Arena, capacity int) *memProbe {
	return &memProbe{
		snaps: make([]memSnapshot, 0, capacity),
		vm:    vm,
		arena: arena,
	}
}

func (p *memProbe) Snapshot(label string) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m) // stops the world briefly — event boundaries only
	as := p.arena.Stats()
	p.snaps = append(p.snaps, memSnapshot{
		label:       label,
		heapLive:    m.HeapAlloc,
		totalAlloc:  m.TotalAlloc,
		mallocs:     m.Mallocs,
		numGC:       m.NumGC,
		lastPause:   m.PauseNs[(m.NumGC+255)%256],
		stackInuse:  m.StackInuse,
		arenaBytes:  as.TotalBytes,
		arenaChunks: as.TotalChunks,
		arenaUsed:   as.CurrentUsed,
		vmStackLen:  len(p.vm.Stack),
		vmStackCap:  cap(p.vm.Stack),
	})
}

// Report prints one row per snapshot. Columns that are windows (heap-Δ, gc,
// pause) are deltas against the previous row; columns that are gauges
// (live-heap, arena, stack) are the value at that instant.
func (p *memProbe) Report(w *strings.Builder) {
	fmt.Fprintf(w, "%-16s %10s %10s %4s %9s %11s %7s %10s %10s\n",
		"event", "heap-Δ", "live-heap", "gc", "pause", "arena-bytes", "chunks", "stack-len", "stack-cap")
	for i, s := range p.snaps {
		heapDelta, gcDelta, pause := "—", "—", "—"
		if i > 0 {
			prev := p.snaps[i-1]
			heapDelta = humanBytes(s.totalAlloc - prev.totalAlloc)
			if n := s.numGC - prev.numGC; n > 0 {
				gcDelta = fmt.Sprintf("%d", n)
				pause = fmt.Sprintf("%.1fµs", float64(s.lastPause)/1e3)
			} else {
				gcDelta = "0"
			}
		}
		fmt.Fprintf(w, "%-16s %10s %10s %4s %9s %11s %7d %10d %10d\n",
			s.label, heapDelta, humanBytes(s.heapLive), gcDelta, pause,
			humanBytes(uint64(s.arenaBytes)), s.arenaChunks, s.vmStackLen, s.vmStackCap)
	}
}

// probeDir is where every probe writes its csv/html output. MEMPROBE_DIR
// (set by the Makefile to an absolute path) overrides the default, which is
// relative to this package dir since go test runs tests there.
func probeDir(t *testing.T) string {
	dir := os.Getenv("MEMPROBE_DIR")
	if dir == "" {
		dir = "../../.memprobe"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func humanBytes(b uint64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1fMB", float64(b)/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1fKB", float64(b)/(1<<10))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

// --- stress sweep with chart output -------------------------------------
//
// TestMemProbeStress sweeps input size (doubling from 10 up to MEMPROBE_MAX,
// default 1M terms) and snapshots each phase at every size. Input size is the
// resolution axis: each phase is one opaque call, so the per-size curve is
// the highest-resolution view available without hooks inside the compiler.
// Output: memprobe.csv (raw rows) and memprobe.html (self-contained SVG
// charts, no external deps) in probeDir — see `make memprobe-view`.

type stressRow struct {
	terms          int
	compileDelta   uint64 // Go-heap bytes allocated during compile
	runDelta       uint64 // Go-heap bytes allocated during run
	compileMallocs uint64 // heap objects allocated (mallocgc calls) during compile
	runMallocs     uint64 // heap objects allocated during run
	liveAfter      uint64 // live heap after run
	gcCompile      uint32 // GC cycles fired during compile
	arenaBytes   int    // arena footprint after compile
	arenaChunks  int
	arenaUsed    int // current-chunk bytes bumped after compile
	stackCap     int // VM stack cap after run (256 unless Push spilled)
}

func TestMemProbeStress(t *testing.T) {
	if os.Getenv("MEMPROBE") == "" {
		t.Skip("set MEMPROBE=1 (or run `make memprobe-stress`) to enable")
	}
	maxTerms := 1_000_000
	if s := os.Getenv("MEMPROBE_MAX"); s != "" {
		fmt.Sscanf(s, "%d", &maxTerms)
	}
	// MEMPROBE_FACTOR controls sweep resolution: growth ratio between sizes.
	// 2.0 (default) doubles; 1.2 gives ~5x more points over the same range.
	factor := 2.0
	if s := os.Getenv("MEMPROBE_FACTOR"); s != "" {
		fmt.Sscanf(s, "%f", &factor)
		if factor <= 1.01 {
			t.Fatalf("MEMPROBE_FACTOR must be > 1.01, got %v", factor)
		}
	}

	var rows []stressRow
	var lastCaps []int // chunk growth history at the largest size, current chunk first
	next := func(n int) int {
		grown := int(float64(n) * factor)
		if grown <= n {
			return n + 1
		}
		return grown
	}
	for n := 10; n <= maxTerms; n = next(n) {
		source := chainedSum(n)
		arena := memory.NewArena(10)
		v := NewVM(arena, false)
		probe := newMemProbe(v, arena, 4)

		probe.Snapshot("start")
		chunk := c.NewChunk(arena)
		compiler := c.NewCompiler(arena, &chunk, source)
		if !compiler.Compile() {
			arena.Free()
			t.Fatal("compile failed")
		}
		probe.Snapshot("compile-done")
		v.Chunk = &chunk
		v.ip = (*uint8)(unsafe.Pointer(&v.Chunk.Code[0]))
		v.Run()
		probe.Snapshot("run-done")

		s0, s1, s2 := probe.snaps[0], probe.snaps[1], probe.snaps[2]
		rows = append(rows, stressRow{
			terms:          n,
			compileDelta:   s1.totalAlloc - s0.totalAlloc,
			runDelta:       s2.totalAlloc - s1.totalAlloc,
			compileMallocs: s1.mallocs - s0.mallocs,
			runMallocs:     s2.mallocs - s1.mallocs,
			liveAfter:      s2.heapLive,
			gcCompile:      s1.numGC - s0.numGC,
			arenaBytes:     s1.arenaBytes,
			arenaChunks:    s1.arenaChunks,
			arenaUsed:      s1.arenaUsed,
			stackCap:       s2.vmStackCap,
		})
		lastCaps = arena.AppendChunkCaps(lastCaps[:0])
		arena.Free()
	}

	dir := probeDir(t)
	writeStressCSV(t, filepath.Join(dir, "memprobe.csv"), rows)
	writeStressHTML(t, filepath.Join(dir, "memprobe.html"), rows, lastCaps)
	t.Logf("wrote %s/memprobe.{csv,html} (%d sizes)", dir, len(rows))
}

// --- arena vs GC head-to-head --------------------------------------------
//
// TestMemProbeArenaVsGC allocates the same object graph two ways — bump-
// allocated in the arena vs individually on the Go heap — and compares what
// the runtime had to do for each: bytes, mallocgc calls, GC cycles, total
// pause. The compile workload can't be used here (the compiler requires the
// arena), so the workload is synthetic: n 64-byte nodes, all retained until
// the end, mimicking a compile phase that keeps its output alive.
// The arena is created INSIDE the measured window on purpose: its chunks are
// themselves Go-heap objects, and hiding that cost would rig the comparison.

type gcNode struct {
	a, b, c, d, e, f, g, h float64 // 64 bytes: one cache line
}

type gcTrialRow struct {
	n        int
	bytes    uint64 // Go-heap bytes allocated during the trial
	mallocs  uint64 // mallocgc calls during the trial
	gcCycles uint32
	pauseNs  uint64 // sum of stop-the-world pause time during the trial
	wallNs   int64
}

func gcTrial(n int, useArena bool, retain []*gcNode) gcTrialRow {
	runtime.GC() // settle: start each trial from a freshly collected heap
	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	start := time.Now()

	if useArena {
		arena := memory.NewArena(1 << 16)
		for i := 0; i < n; i++ {
			p := memory.Alloc[gcNode](arena)
			p.a = float64(i)
			retain[i] = p
		}
		defer arena.Free()
	} else {
		for i := 0; i < n; i++ {
			p := &gcNode{a: float64(i)}
			retain[i] = p
		}
	}

	wall := time.Since(start)
	runtime.ReadMemStats(&m1)
	return gcTrialRow{
		n:        n,
		bytes:    m1.TotalAlloc - m0.TotalAlloc,
		mallocs:  m1.Mallocs - m0.Mallocs,
		gcCycles: m1.NumGC - m0.NumGC,
		pauseNs:  m1.PauseTotalNs - m0.PauseTotalNs,
		wallNs:   wall.Nanoseconds(),
	}
}

func TestMemProbeArenaVsGC(t *testing.T) {
	if os.Getenv("MEMPROBE") == "" {
		t.Skip("set MEMPROBE=1 (or run `make memprobe-arena-gc`) to enable")
	}
	maxN := 1_000_000
	if s := os.Getenv("MEMPROBE_MAX"); s != "" {
		fmt.Sscanf(s, "%d", &maxN)
	}

	// retention slice preallocated once, outside every measured window, so
	// only the strategies' own allocations differ between trials
	retain := make([]*gcNode, maxN)
	var arenaRows, heapRows []gcTrialRow
	for n := 1000; n <= maxN; n *= 2 {
		arenaRows = append(arenaRows, gcTrial(n, true, retain))
		heapRows = append(heapRows, gcTrial(n, false, retain))
	}

	dir := probeDir(t)
	var csv strings.Builder
	csv.WriteString("n,strategy,heap_bytes,mallocs,gc_cycles,pause_ns,wall_ns\n")
	for i := range arenaRows {
		a, h := arenaRows[i], heapRows[i]
		fmt.Fprintf(&csv, "%d,arena,%d,%d,%d,%d,%d\n", a.n, a.bytes, a.mallocs, a.gcCycles, a.pauseNs, a.wallNs)
		fmt.Fprintf(&csv, "%d,heap,%d,%d,%d,%d,%d\n", h.n, h.bytes, h.mallocs, h.gcCycles, h.pauseNs, h.wallNs)
	}
	if err := os.WriteFile(filepath.Join(dir, "memprobe-arena-gc.csv"), []byte(csv.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	writeArenaVsGCHTML(t, filepath.Join(dir, "memprobe-arena-gc.html"), arenaRows, heapRows)
	t.Logf("wrote %s/memprobe-arena-gc.{csv,html} (%d sizes)", dir, len(arenaRows))
}

func writeArenaVsGCHTML(t *testing.T, path string, arenaRows, heapRows []gcTrialRow) {
	xs := make([]float64, len(arenaRows))
	pick := func(rows []gcTrialRow, f func(gcTrialRow) float64) []float64 {
		ys := make([]float64, len(rows))
		for i, r := range rows {
			ys[i] = f(r)
		}
		return ys
	}
	for i, r := range arenaRows {
		xs[i] = mathLog2(float64(r.n))
	}
	var sb strings.Builder
	caption := func(text string) {
		fmt.Fprintf(&sb, `<p style="color:#555;font-size:13px;margin:0 0 20px 0">%s</p>`, text)
	}
	sb.WriteString(`<meta charset="utf-8"><title>memprobe arena vs GC</title><body style="font-family:monospace;max-width:860px;margin:20px auto">`)
	sb.WriteString(`<h2>arena vs GC head-to-head</h2><p>Same workload — n 64-byte nodes, all
retained — allocated via the arena (green) vs one-object-at-a-time on the Go heap (purple).
X: n (log). The arena's own chunk allocations are counted, deliberately.</p>`)

	sb.WriteString(svgChart("mallocgc calls", "", xs, []series{
		{"arena", "#27ae60", pick(arenaRows, func(r gcTrialRow) float64 { return float64(r.mallocs) })},
		{"heap", "#8e44ad", pick(heapRows, func(r gcTrialRow) float64 { return float64(r.mallocs) })}}, true, nil))
	caption(`The core structural difference: heap does one mallocgc per node (slope 1), the arena
does one per <i>chunk</i> (a handful, growing only with doublings). Everything below follows
from this line.`)

	sb.WriteString(svgChart("Go-heap bytes allocated", "B", xs, []series{
		{"arena", "#27ae60", pick(arenaRows, func(r gcTrialRow) float64 { return float64(r.bytes) })},
		{"heap", "#8e44ad", pick(heapRows, func(r gcTrialRow) float64 { return float64(r.bytes) })}}, true, nil))
	caption(`Bytes are similar-ish (the nodes have to live somewhere) — the arena may even
overshoot from chunk doubling. If you expected the arena to save bytes, this chart is the
correction: arenas save <i>objects</i> and <i>lifetimes</i>, not bytes.`)

	sb.WriteString(svgChart("GC cycles / total STW pause (µs)", "", xs, []series{
		{"arena gc cycles", "#27ae60", pick(arenaRows, func(r gcTrialRow) float64 { return float64(r.gcCycles) })},
		{"heap gc cycles", "#8e44ad", pick(heapRows, func(r gcTrialRow) float64 { return float64(r.gcCycles) })},
		{"arena pause µs", "#16a085", pick(arenaRows, func(r gcTrialRow) float64 { return float64(r.pauseNs) / 1e3 })},
		{"heap pause µs", "#c0392b", pick(heapRows, func(r gcTrialRow) float64 { return float64(r.pauseNs) / 1e3 })}}, true, nil))
	caption(`What the GC actually did during each trial. Mark cost scales with live object
<i>count</i>: a million retained heap nodes are a million pointers to trace; the same data in
arena chunks is a handful of large objects with no interior pointers for the GC to chase.`)

	sb.WriteString(svgChart("wall time (µs)", "", xs, []series{
		{"arena", "#27ae60", pick(arenaRows, func(r gcTrialRow) float64 { return float64(r.wallNs) / 1e3 })},
		{"heap", "#8e44ad", pick(heapRows, func(r gcTrialRow) float64 { return float64(r.wallNs) / 1e3 })}}, true, nil))
	caption(`Single-shot wall time — indicative only, NOT a benchmark (one sample, no
distribution; use the arena_bench_test.go pairs with benchstat for defensible speed claims).
Shown because allocation cost and GC assist time land here in ways the counters above explain.`)

	sb.WriteString(`</body>`)
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestMemProbeArena holds the workload fixed (MEMPROBE_TERMS, default 1M)
// and sweeps the arena's *initial capacity* instead — 1KB doubling up to
// MEMPROBE_ARENA_MAX (default 8MB). This is the experiment NewArena's doc
// comment asks for ("size it to fit your typical workload"): it shows what
// initial sizing actually buys — fewer grow() chunks — and what it cannot
// buy, which the compile-heap-bytes line will reveal.
func TestMemProbeArena(t *testing.T) {
	if os.Getenv("MEMPROBE") == "" {
		t.Skip("set MEMPROBE=1 (or run `make memprobe-arena`) to enable")
	}
	terms := 1_000_000
	if s := os.Getenv("MEMPROBE_TERMS"); s != "" {
		fmt.Sscanf(s, "%d", &terms)
	}
	maxCap := 8 << 20
	if s := os.Getenv("MEMPROBE_ARENA_MAX"); s != "" {
		fmt.Sscanf(s, "%d", &maxCap)
	}

	source := chainedSum(terms)
	var rows []stressRow
	var caps []int
	for capBytes := 1 << 10; capBytes <= maxCap; capBytes *= 2 {
		arena := memory.NewArena(capBytes)
		v := NewVM(arena, false)
		probe := newMemProbe(v, arena, 4)

		probe.Snapshot("start")
		chunk := c.NewChunk(arena)
		compiler := c.NewCompiler(arena, &chunk, source)
		if !compiler.Compile() {
			arena.Free()
			t.Fatal("compile failed")
		}
		probe.Snapshot("compile-done")
		v.Chunk = &chunk
		v.ip = (*uint8)(unsafe.Pointer(&v.Chunk.Code[0]))
		v.Run()
		probe.Snapshot("run-done")

		s0, s1, s2 := probe.snaps[0], probe.snaps[1], probe.snaps[2]
		caps = append(caps, capBytes)
		rows = append(rows, stressRow{
			terms:          capBytes, // x-axis carries initial cap in this sweep
			compileDelta:   s1.totalAlloc - s0.totalAlloc,
			runDelta:       s2.totalAlloc - s1.totalAlloc,
			compileMallocs: s1.mallocs - s0.mallocs,
			runMallocs:     s2.mallocs - s1.mallocs,
			liveAfter:      s2.heapLive,
			gcCompile:      s1.numGC - s0.numGC,
			arenaBytes:     s1.arenaBytes,
			arenaChunks:    s1.arenaChunks,
			arenaUsed:      s1.arenaUsed,
			stackCap:       s2.vmStackCap,
		})
		arena.Free()
	}

	dir := probeDir(t)
	writeArenaCSV(t, filepath.Join(dir, "memprobe-arena.csv"), rows)
	writeArenaHTML(t, filepath.Join(dir, "memprobe-arena.html"), rows, terms)
	t.Logf("wrote %s/memprobe-arena.{csv,html} (%d caps, workload %d terms)", dir, len(caps), terms)
}

func writeArenaCSV(t *testing.T, path string, rows []stressRow) {
	var sb strings.Builder
	sb.WriteString("initial_cap_bytes,compile_heap_bytes,run_heap_bytes,compile_mallocs,run_mallocs,live_heap_after,gc_cycles_compile,arena_bytes,arena_chunks,arena_current_used,vm_stack_cap\n")
	for _, r := range rows {
		fmt.Fprintf(&sb, "%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
			r.terms, r.compileDelta, r.runDelta, r.compileMallocs, r.runMallocs,
			r.liveAfter, r.gcCompile, r.arenaBytes, r.arenaChunks, r.arenaUsed, r.stackCap)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeArenaHTML(t *testing.T, path string, rows []stressRow, terms int) {
	xs := make([]float64, len(rows))
	heapC := make([]float64, len(rows))
	malC := make([]float64, len(rows))
	arenaB := make([]float64, len(rows))
	gcs := make([]float64, len(rows))
	chks := make([]float64, len(rows))
	for i, r := range rows {
		xs[i] = mathLog2(float64(r.terms)) // terms field carries initial cap here
		heapC[i] = float64(r.compileDelta)
		malC[i] = float64(r.compileMallocs)
		arenaB[i] = float64(r.arenaBytes)
		gcs[i] = float64(r.gcCompile)
		chks[i] = float64(r.arenaChunks)
	}
	bytesLabel := func(v float64) string { return humanBytes(uint64(mathPow2(v))) }
	var sb strings.Builder
	caption := func(text string) {
		fmt.Fprintf(&sb, `<p style="color:#555;font-size:13px;margin:0 0 20px 0">%s</p>`, text)
	}
	sb.WriteString(`<meta charset="utf-8"><title>memprobe arena sizing</title><body style="font-family:monospace;max-width:860px;margin:20px auto">`)
	fmt.Fprintf(&sb, `<h2>arena initial-capacity sweep</h2><p>Workload fixed at %s terms; X is
NewArena's initial capacity (log scale, doubling). The question: what does initial sizing buy?</p>`,
		humanCount(terms))

	sb.WriteString(svgChart("Go-heap bytes + objects allocated during compile", "B", xs, []series{
		{"compile heap bytes", "#c0392b", heapC}, {"compile mallocs", "#2980b9", malC}}, true, bytesLabel))
	caption(`If these lines stay flat as initial capacity grows, a bigger arena is NOT reducing
Go-heap traffic — meaning the spill doesn't come from the arena running out of room, and no
amount of pre-sizing fixes it. Ask where the escaping allocations actually come from.`)

	sb.WriteString(svgChart("arena chunks / GC cycles during compile", "", xs, []series{
		{"arena chunks", "#27ae60", chks}, {"gc cycles", "#c0392b", gcs}}, false, bytesLabel))
	caption(`What sizing DOES buy: chunk count should fall toward 1 once the initial capacity
covers the workload's arena demand — each avoided chunk is one avoided grow() (a make + list
link). Where the green line hits 1 is your right-sized arena for this workload.`)

	sb.WriteString(svgChart("total arena capacity after compile", "B", xs, []series{
		{"arena capacity", "#27ae60", arenaB}}, true, bytesLabel))
	caption(`Total capacity vs the initial cap you asked for. Because grow() doubles from the
current chunkCap, an undersized arena can overshoot: many doublings can reserve more total
memory than one right-sized chunk would have. The minimum of this curve is the cheapest
configuration.`)

	sb.WriteString(`</body>`)
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeStressCSV(t *testing.T, path string, rows []stressRow) {
	var sb strings.Builder
	sb.WriteString("terms,compile_heap_bytes,run_heap_bytes,compile_mallocs,run_mallocs,live_heap_after,gc_cycles_compile,arena_bytes,arena_chunks,arena_current_used,vm_stack_cap\n")
	for _, r := range rows {
		fmt.Fprintf(&sb, "%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
			r.terms, r.compileDelta, r.runDelta, r.compileMallocs, r.runMallocs,
			r.liveAfter, r.gcCompile, r.arenaBytes, r.arenaChunks, r.arenaUsed, r.stackCap)
	}
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

// series is one polyline on a chart.
type series struct {
	name  string
	color string
	ys    []float64
}

// svgChart renders series against log2(terms) on X. logY switches the Y axis
// to log10 (bytes span orders of magnitude; GC counts stay linear).
func svgChart(title, yUnit string, xs []float64, ss []series, logY bool, xLabel func(float64) string) string {
	if xLabel == nil {
		xLabel = func(v float64) string { return humanCount(int(mathPow2(v))) }
	}
	const w, h = 820, 300
	const left, right, top, bottom = 70.0, 790.0, 30.0, 260.0
	xMin, xMax := xs[0], xs[len(xs)-1]
	yMax := 1.0
	for _, s := range ss {
		for _, y := range s.ys {
			if y > yMax {
				yMax = y
			}
		}
	}
	yVal := func(y float64) float64 {
		if logY {
			if y < 1 {
				y = 1
			}
			return bottom - (bottom-top)*(mathLog10(y)/mathLog10(yMax))
		}
		return bottom - (bottom-top)*(y/yMax)
	}
	xVal := func(x float64) float64 {
		return left + (right-left)*((x-xMin)/(xMax-xMin))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, `<svg viewBox="0 0 %d %d" style="max-width:%dpx;background:#fff;border:1px solid #ddd;margin:8px 0">`, w, h, w)
	fmt.Fprintf(&sb, `<text x="%f" y="20" font-size="14" font-family="monospace">%s</text>`, left, title)
	// axes
	fmt.Fprintf(&sb, `<line x1="%f" y1="%f" x2="%f" y2="%f" stroke="#333"/>`, left, bottom, right, bottom)
	fmt.Fprintf(&sb, `<line x1="%f" y1="%f" x2="%f" y2="%f" stroke="#333"/>`, left, top, left, bottom)
	// x tick per point (terms doubles each step; label every 4th)
	for i, x := range xs {
		px := xVal(x)
		fmt.Fprintf(&sb, `<line x1="%f" y1="%f" x2="%f" y2="%f" stroke="#333"/>`, px, bottom, px, bottom+4)
		if i%4 == 0 || i == len(xs)-1 {
			fmt.Fprintf(&sb, `<text x="%f" y="%f" font-size="10" font-family="monospace" text-anchor="middle">%s</text>`,
				px, bottom+18, xLabel(x))
		}
	}
	// y max label + unit
	fmt.Fprintf(&sb, `<text x="%f" y="%f" font-size="10" font-family="monospace" text-anchor="end">%s</text>`, left-6, top+4, humanBytesOrCount(yMax, yUnit))
	fmt.Fprintf(&sb, `<text x="%f" y="%f" font-size="10" font-family="monospace" text-anchor="end">0</text>`, left-6, bottom+4)
	// series
	for si, s := range ss {
		var pts strings.Builder
		for i, y := range s.ys {
			fmt.Fprintf(&pts, "%f,%f ", xVal(xs[i]), yVal(y))
		}
		fmt.Fprintf(&sb, `<polyline points="%s" fill="none" stroke="%s" stroke-width="2"/>`, pts.String(), s.color)
		fmt.Fprintf(&sb, `<text x="%f" y="%f" font-size="11" font-family="monospace" fill="%s">%s</text>`,
			left+10, top+14+float64(si)*14, s.color, s.name)
	}
	sb.WriteString(`</svg>`)
	return sb.String()
}

func writeStressHTML(t *testing.T, path string, rows []stressRow, lastCaps []int) {
	xs := make([]float64, len(rows))
	heapC := make([]float64, len(rows))
	heapR := make([]float64, len(rows))
	malC := make([]float64, len(rows))
	malR := make([]float64, len(rows))
	arenaB := make([]float64, len(rows))
	arenaU := make([]float64, len(rows))
	liveA := make([]float64, len(rows))
	gcs := make([]float64, len(rows))
	chks := make([]float64, len(rows))
	for i, r := range rows {
		xs[i] = mathLog2(float64(r.terms))
		heapC[i] = float64(r.compileDelta)
		heapR[i] = float64(r.runDelta)
		malC[i] = float64(r.compileMallocs)
		malR[i] = float64(r.runMallocs)
		arenaB[i] = float64(r.arenaBytes)
		arenaU[i] = float64(r.arenaUsed)
		liveA[i] = float64(r.liveAfter)
		gcs[i] = float64(r.gcCompile)
		chks[i] = float64(r.arenaChunks)
	}
	var sb strings.Builder
	caption := func(text string) {
		fmt.Fprintf(&sb, `<p style="color:#555;font-size:13px;margin:0 0 20px 0">%s</p>`, text)
	}
	sb.WriteString(`<meta charset="utf-8"><title>memprobe</title><body style="font-family:monospace;max-width:860px;margin:20px auto">`)
	sb.WriteString(`<h2>memprobe stress sweep</h2>`)
	sb.WriteString(`<p>Each x-position is one complete fresh compile+run at that input size (not a
timeline). X is terms on a log scale; bytes/count charts use log-Y, so a straight line means
power-law scaling and a <b>kink</b> means the allocator changed regime at that size — kinks are
what to look for.</p>`)

	sb.WriteString(svgChart("Go-heap bytes allocated per phase", "B", xs, []series{
		{"compile", "#c0392b", heapC}, {"run", "#2980b9", heapR}}, true, nil))
	caption(`Bytes that escaped the arena onto Go's GC-managed heap (MemStats.TotalAlloc diffed
around each phase). Arena-served allocations don't appear here at all — that's the point: this
line is the arena's <i>leakage</i>. Expect compile to grow with size (Chunk.Code/Consts/Lines
spilling via append past their arena capacity) and run to sit near zero unless the VM stack
outgrows its 256-slot arena backing.`)

	sb.WriteString(svgChart("arena footprint after compile vs live Go-heap after run", "B", xs, []series{
		{"arena capacity", "#27ae60", arenaB}, {"current-chunk used", "#e67e22", arenaU},
		{"live Go-heap", "#8e44ad", liveA}}, true, nil))
	caption(`Who owns the working set: total arena chunk capacity, bytes actually bumped in the
<i>current</i> chunk (old chunks' usage is unrecorded — their tails were stranded by grow, so
whole-arena utilization isn't knowable from stats), and bytes still live on the Go heap after
run (HeapAlloc). If purple converges with or overtakes green as size grows, the GC owns most of
the working set despite the arena. A big gap between green and orange means capacity far ahead
of current-chunk use — either stranded tails or a freshly doubled chunk.`)
	if len(lastCaps) > 0 {
		var caps strings.Builder
		for i := len(lastCaps) - 1; i >= 0; i-- { // print oldest→newest
			if i < len(lastCaps)-1 {
				caps.WriteString(" → ")
			}
			caps.WriteString(humanBytes(uint64(lastCaps[i])))
		}
		caption(fmt.Sprintf(`Chunk growth history at the largest size (oldest → newest):
<b>%s</b> — this is grow()'s policy (double, or minSize if larger) made visible.`, caps.String()))
	}

	sb.WriteString(svgChart("heap objects allocated (mallocgc calls) per phase", "", xs, []series{
		{"compile mallocs", "#c0392b", malC}, {"run mallocs", "#2980b9", malR}}, true, nil))
	caption(`Object <i>count</i>, not bytes (MemStats.Mallocs diffed per phase). Read against the
bytes chart: few objects + many bytes = slice-doubling (cheap for GC); many objects + few bytes
= per-node garbage (expensive — GC mark cost scales with object count, not bytes). Stair-steps
here are append's growth policy firing.`)

	sb.WriteString(svgChart("GC cycles during compile / arena chunk count", "", xs, []series{
		{"gc cycles", "#c0392b", gcs}, {"arena chunks", "#27ae60", chks}}, false, nil))
	caption(`Linear Y. GC cycles = full mark+sweep collections triggered during compile (fires
when heap grows past GOGC's target, default +100% since last cycle). Arena chunks = how many
times the arena had to grow a new chunk. Both are counts of "the allocator ran out and did
something expensive" — compare where each starts climbing.`)

	sb.WriteString(`</body>`)
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mathLog10(x float64) float64 { return math.Log10(x) }
func mathLog2(x float64) float64  { return math.Log2(x) }
func mathPow2(x float64) float64  { return math.Pow(2, x) }

func humanCount(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%dk", n/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func humanBytesOrCount(v float64, unit string) string {
	if unit == "B" {
		return humanBytes(uint64(v))
	}
	return fmt.Sprintf("%.0f", v)
}

// TestMemProbe is an instrumented run, not a benchmark: one execution per
// input size, snapshotted at each phase boundary. Gated behind MEMPROBE=1 so
// plain `make test` stays clean. Run via `make memprobe`.
func TestMemProbe(t *testing.T) {
	if os.Getenv("MEMPROBE") == "" {
		t.Skip("set MEMPROBE=1 (or run `make memprobe`) to enable")
	}

	for _, n := range []int{10, 10_000, 1_000_000} {
		t.Run(fmt.Sprintf("terms=%d", n), func(t *testing.T) {
			source := chainedSum(n)

			arena := memory.NewArena(10)
			v := NewVM(arena, false)
			probe := newMemProbe(v, arena, 8)

			probe.Snapshot("start")

			chunk := c.NewChunk(arena)
			compiler := c.NewCompiler(arena, &chunk, source)
			if !compiler.Compile() {
				arena.Free()
				t.Fatal("compile failed")
			}
			probe.Snapshot("compile-done")

			v.Chunk = &chunk
			v.ip = (*uint8)(unsafe.Pointer(&v.Chunk.Code[0]))
			v.Run()
			probe.Snapshot("run-done")

			arena.Free()
			probe.Snapshot("arena-freed")

			var sb strings.Builder
			sb.WriteByte('\n')
			probe.Report(&sb)
			t.Log(sb.String())
		})
	}
}
