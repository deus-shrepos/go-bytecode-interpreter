package source

import (
	"fmt"
	"go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"os"
)

func ReadFile(path string) ([]byte, int) {
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Glox: %s", err)
		return nil, 1
	}
	return source, 1
}

func CompileProgram(path string, trace uint8) int {
	arena := memory.NewArena(1 << 20)
	source := readProgram(path)
	chunk := compiler.NewChunk(arena)
	c := compiler.NewCompiler(arena, &chunk, source)
	if trace != 0 {
		c.SetTrace(trace)
	}
	if !c.Compile() {
		return 1
	}
	return 0
}

func RunProgram(source string) {}

func readProgram(path string) []byte {
	source, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Glox: %s", err)
		os.Exit(1)
	}

	if len(source) == 0 {
		return nil
	}
	return source
}
