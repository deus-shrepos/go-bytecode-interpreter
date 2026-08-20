BINARY := glox
CMD := ./cmd/glox
MEMORY_PKG := ./internals/memory
PACKAGES := compiler debug errors lexer memory repl value vm

.PHONY: all build run test test-verbose test-race bench bench-memory vet fmt clean $(addprefix test-,$(PACKAGES)) test-pkg

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
