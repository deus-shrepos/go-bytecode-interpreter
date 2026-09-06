BINARY := glox
CMD := ./cmd/glox
MEMORY_PKG := ./internals/memory
VM_PKG := ./internals/vm
PACKAGES := compiler debug errors lexer memory repl value vm

PPROF_DIR := .pprof
PPROF_PORT := 6060
STRESS_BENCHTIME := 20x
BASELINE_DIR := .bench
BENCH_COUNT := 10
MEMPROBE_DIR := .memprobe

.PHONY: all build run test test-verbose test-race bench bench-memory vet fmt clean $(addprefix test-,$(PACKAGES)) test-pkg \
	bench-stress gctrace-stress pprof-heap pprof-view clean-pprof \
	bench-baseline bench-compare escape-analysis clean-bench

all: build

build:
	go build -o $(BINARY) $(CMD)

run:
	go run $(CMD) $(ARGS)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -race ./...

$(addprefix test-,$(PACKAGES)):
	go test -v ./internals/$(patsubst test-%,%,$@)

test-pkg:
	go test -v $(PKG)

bench:
	go test -bench=. -benchmem ./...

bench-memory:
	go test -bench=. -benchmem $(MEMORY_PKG)

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f $(BINARY)

# --- Profiling: internals/vm/vm_bench_test.go's BenchmarkVM_Stress ---
# See .claude/worklog for how these came about: comparing arena bump-pointer
# allocation against ordinary GC-heap growth (Chunk.Code/Consts/Lines
# outgrowing their initial 64-slot arena capacity via plain append).

bench-stress: ## Run the stress benchmark table (10 .. 1,000,000 terms) with alloc stats.
	go test -bench=BenchmarkVM_Stress -benchmem $(VM_PKG)

gctrace-stress: ## Show real GC cycles firing during the 1,000,000-term case.
	@mkdir -p $(PPROF_DIR)
	go test -c -o $(PPROF_DIR)/vm.test $(VM_PKG)
		-test.bench='BenchmarkVM_Stress/terms=1000000$$' -test.benchtime=$(STRESS_BENCHTIME)
	GODEBUG=gctrace=1 $(PPROF_DIR)/vm.test -test.run=xxx \

pprof-heap: ## Build a pinned test binary and capture a full-resolution heap profile.
	@mkdir -p $(PPROF_DIR)
	go test -c -o $(PPROF_DIR)/vm.test $(VM_PKG)
	$(PPROF_DIR)/vm.test -test.run=xxx -test.bench=BenchmarkVM_Stress \
		-test.benchtime=$(STRESS_BENCHTIME) \
		-test.memprofile=$(PPROF_DIR)/heap.prof -test.memprofilerate=1
	@echo "Profile written to $(PPROF_DIR)/heap.prof — run 'make pprof-view' to open it."

pprof-view: ## Serve the last captured heap profile in pprof's web UI (flame graph, source view).
	@test -f $(PPROF_DIR)/heap.prof || (echo "No profile yet — run 'make pprof-heap' first." && exit 1)
	go tool pprof -http=:$(PPROF_PORT) -no_browser $(PPROF_DIR)/vm.test $(PPROF_DIR)/heap.prof

clean-pprof:
	rm -rf $(PPROF_DIR)

# --- Statistical comparison (benchstat) and escape analysis ---
# benchstat needs several independent samples, not one long run: -count=N
# reruns each benchmark N times rather than growing -benchtime. Requires
# `go install golang.org/x/perf/cmd/benchstat@latest`.

bench-baseline: ## Capture N samples of BenchmarkVM_Stress as the comparison baseline.
	@mkdir -p $(BASELINE_DIR)
	go test -bench=BenchmarkVM_Stress -benchmem -count=$(BENCH_COUNT) $(VM_PKG) | tee $(BASELINE_DIR)/baseline.txt

bench-compare: ## Capture N current samples and diff them against the saved baseline.
	@test -f $(BASELINE_DIR)/baseline.txt || (echo "No baseline yet — run 'make bench-baseline' first." && exit 1)
	@mkdir -p $(BASELINE_DIR)
	go test -bench=BenchmarkVM_Stress -benchmem -count=$(BENCH_COUNT) $(VM_PKG) | tee $(BASELINE_DIR)/current.txt
	go tool benchstat $(BASELINE_DIR)/baseline.txt $(BASELINE_DIR)/current.txt

clean-bench:
	rm -rf $(BASELINE_DIR)

memprobe: ## Instrumented single runs: per-phase heap/GC/arena/VM-stack snapshot table.
	MEMPROBE=1 go test -run 'TestMemProbe$$' -v $(VM_PKG)

memprobe-stress: ## Sweep sizes 10..MEMPROBE_MAX (default 1M), emit $(MEMPROBE_DIR)/memprobe.{csv,html}.
	MEMPROBE=1 MEMPROBE_DIR=$(CURDIR)/$(MEMPROBE_DIR) go test -run 'TestMemProbeStress$$' -v $(VM_PKG)

memprobe-arena: ## Fixed workload (MEMPROBE_TERMS, default 1M), sweep NewArena initial cap 1KB..MEMPROBE_ARENA_MAX.
	MEMPROBE=1 MEMPROBE_DIR=$(CURDIR)/$(MEMPROBE_DIR) go test -run 'TestMemProbeArena$$' -v $(VM_PKG)

memprobe-arena-gc: ## Head-to-head: n retained nodes allocated via arena vs Go heap; GC cycles, pause, mallocs.
	MEMPROBE=1 MEMPROBE_DIR=$(CURDIR)/$(MEMPROBE_DIR) go test -run 'TestMemProbeArenaVsGC$$' -v $(VM_PKG)

memprobe-view: ## Serve all probe charts on localhost (sandboxed browsers can't read dot-dirs via file://).
	@test -d $(MEMPROBE_DIR) || (echo "No probe output yet — run a memprobe-* target first." && exit 1)
	@echo "Charts at http://localhost:$(PPROF_PORT)/ — Ctrl+C to stop."
	@cd $(MEMPROBE_DIR) && python3 -m http.server $(PPROF_PORT)

clean-memprobe:
	rm -rf $(MEMPROBE_DIR)

escape-analysis: ## Show compiler escape-analysis decisions for the memory + vm packages.
	@mkdir -p $(PPROF_DIR)
	go build -gcflags='-m -m' $(MEMORY_PKG) $(VM_PKG) 2>&1 | tee $(PPROF_DIR)/escape.txt
	@echo "Escape analysis written to $(PPROF_DIR)/escape.txt"
