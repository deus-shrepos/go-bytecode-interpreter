package compiler

import (
	"go-bytecode-interpter/internals/lexer"
)

type Compiler struct{}

func NewCompiler() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(source []byte) {
	scanner := lexer.NewScanner(source)
	scanner.ScanToken() // start the scanner?

}
