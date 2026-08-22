package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/errors"
	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/memory"
	"os"
)

type Compiler struct {
	current   lexer.Token
	previous  lexer.Token
	scanner   lexer.Scanner
	hadError  bool
	panicMode bool
	chunk     *Chunk
	arena     *memory.Arena // long-lived objects
}

func NewCompiler(arena *memory.Arena, chunk *Chunk, source []byte) *Compiler {
	return &Compiler{
		arena:    arena,
		chunk:    chunk,
		scanner:  *lexer.NewScanner(source),
		hadError: false,
	}
}

func (c *Compiler) Compile() bool {
	c.advance()
	c.expression()
	c.consume(lexer.EOF, "Expect end of expression.")
	return !c.hadError
}

func (c *Compiler) advance() {
	c.previous = c.current
	for {
		c.current = c.scanner.ScanToken()
		if c.current.Type != lexer.ERROR {
			break // move the scanner
		}

		// if in panic mode, compile as normal
		// i.e resynchronise
		if c.panicMode {
			return
		}

		c.reportError(c.current.Lexeme())
	}
}

func (c *Compiler) expression() {

}

func (c *Compiler) consume(tokeType lexer.TokenType, message string) {
	if c.current.Type == tokeType {
		c.advance()
		return
	}
	c.reportError(message)
}

func (c *Compiler) EmitByteCode(byteCode byte) {

}

func (c *Compiler) reportError(message string) {
	c.panicMode = true

	fmt.Fprintf(
		os.Stderr,
		"%s",
		errors.NewError(c.current, errors.CompilePhaseError, message),
	)
	c.hadError = true
}
