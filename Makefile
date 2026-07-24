BINARY := glox
CMD := ./cmd/glox
MEMORY_PKG := ./internals/memory

.PHONY: all build run test test-verbose test-race bench bench-memory vet fmt clean

all: build

build:
	go build -o $(BINARY) $(CMD)

run:
	go run $(CMD)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -race ./...

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
