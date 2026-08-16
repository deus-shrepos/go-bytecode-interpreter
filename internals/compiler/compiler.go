package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/lexer"
)

type Compiler struct {
	Chunks Chunk
}

func NewCompiler() *Compiler {
	return &Compiler{}
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
		}
	}
}
