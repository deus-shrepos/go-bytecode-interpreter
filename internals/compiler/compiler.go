package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/memory"
)

type Compiler struct {
	Chunks Chunk
	arena  *memory.Arena // long-lived objects
}

func NewCompiler(arena *memory.Arena) *Compiler {
	return &Compiler{
		arena: arena,
	}
}

func (c *Compiler) Compile(source []byte) {
	scanner := lexer.NewScanner(source)
	for {
		token := scanner.ScanToken()
		if token.Type == lexer.EOF {
			// scanning done so we are all good
			return // just for now
		}

		if token.Type == lexer.ERROR {
			fmt.Printf("Scanner Error: %v", token.GetLexme())
			return
		}
		fmt.Println(token)
	}
}
