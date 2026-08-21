package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/errors"
	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/memory"
	"os"
)

type Compiler struct {
	current  lexer.Token
	previous lexer.Token
	scanner  lexer.Scanner
	chunk    *Chunk
	arena    *memory.Arena // long-lived objects
}

func NewCompiler(arena *memory.Arena, chunk *Chunk, source []byte) *Compiler {
	return &Compiler{
		arena:   arena,
		chunk:   chunk,
		scanner: *lexer.NewScanner(source),
	}
}

func (c *Compiler) Compile() bool {
	c.advance()
	c.expression()
	c.consume(lexer.EOF, "Expect end of expression.")
	return false
}

func (c *Compiler) advance() {
	c.previous = c.current
	for {
		c.current = c.scanner.ScanToken()
		if c.current.Type != lexer.ERROR {
			break // move the scanner
		}

		err := errors.NewError(c.current, errors.CompilePhaseError, c.current.GetLexme())
		fmt.Fprintf(os.Stderr, "%s", err)
	}
}

func (c *Compiler) expression() {

}

func (c *Compiler) consume(tokeType lexer.TokenType, message string) {

}
