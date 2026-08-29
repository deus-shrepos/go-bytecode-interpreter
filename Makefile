BINARY := glox
CMD := ./cmd/glox
MEMORY_PKG := ./internals/memory
VM_PKG := ./internals/vm
PACKAGES := compiler debug errors lexer memory repl value vm

PPROF_DIR := .pprof
PPROF_PORT := 6060
STRESS_BENCHTIME := 20x

.PHONY: all build run test test-verbose test-race bench bench-memory vet fmt clean $(addprefix test-,$(PACKAGES)) test-pkg \
	bench-stress gctrace-stress pprof-heap pprof-view clean-pprof

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
	GODEBUG=gctrace=1 $(PPROF_DIR)/vm.test -test.run=xxx \
		-test.bench='BenchmarkVM_Stress/terms=1000000$$' -test.benchtime=$(STRESS_BENCHTIME)

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
