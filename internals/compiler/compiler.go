package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/errors"
	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/memory"
	"math"
	"os"
	"strconv"
)

type Compiler struct {
	current        lexer.Token
	previous       lexer.Token
	scanner        lexer.Scanner
	hadError       bool
	panicMode      bool
	debugMode      bool
	chunk          *Chunk
	compilingChunk *Chunk
	arena          *memory.Arena // long-lived objects
}

func NewCompiler(arena *memory.Arena, chunk *Chunk, source []byte) *Compiler {
	return &Compiler{
		arena:     arena,
		chunk:     chunk,
		scanner:   *lexer.NewScanner(source),
		hadError:  false,
		panicMode: false,
		debugMode: false,
	}
}

func (c *Compiler) Compile() bool {
	c.advance()
	c.expression()
	c.consume(lexer.EOF, "Expect end of expression.")
	c.endCompiler()
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
	c.parsePrecedence(PrecAssignment)
}

func (c *Compiler) consume(tokeType lexer.TokenType, message string) {
	if c.current.Type == tokeType {
		c.advance()
		return
	}
	c.reportError(message)
}

func (c *Compiler) emitByteCode(byteCode OpCode) {
	// write the chunk and record the previous line for runtime error info
	c.chunk.WriteChunk(byteCode, c.previous.Line)
}

func (c *Compiler) endCompiler() {
	c.emitByteCode(OP_RETURN)
	if c.debugMode && c.hadError {
		DisassembleChunk(*c.chunk, "code")
	}
}

func (c *Compiler) emitReturn() {
	c.emitByteCode(OP_RETURN)
}

func (c *Compiler) emitBytes(b1 OpCode, b2 OpCode) {
	c.emitByteCode(b1)
	c.emitByteCode(b2)
}

func (c *Compiler) grouping() {
	c.expression()
	c.consume(lexer.RIGHT_PAREN, "Expect ')' after expression.")
}

func (c *Compiler) parsePrecedence(prec Precedence) {
	c.advance() // consumer the prefix
	prefixRule := rules[c.previous.Type].prefix
	if prefixRule == nil {
		c.reportError("Expect Expression.")
		return
	}
	prefixRule(c)
	for prec <= rules[c.current.Type].prec {
		c.advance()
		infixRule := rules[c.previous.Type].infix
		infixRule(c)
	}
}

func (c *Compiler) unary() {
	operaterType := c.previous.Type

	// compile the operand
	c.parsePrecedence(PrecUnary)
	switch operaterType {
	case lexer.MINUS:
		c.emitByteCode(OP_NEGATE)
	default:
		return
	}

}

func (c *Compiler) binary() {
	operatorType := c.previous.Type

	rule := rules[operatorType]
	// left-assoc precedence
	c.parsePrecedence(Precedence(rule.prec + 1))

	switch operatorType {
	case lexer.PLUS:
		c.emitByteCode(OP_ADD)
	case lexer.MINUS:
		c.emitByteCode(OP_SUBTRACT)
	case lexer.STAR:
		c.emitByteCode(OP_MULTIPLY)
	case lexer.SLASH:
		c.emitByteCode(OP_DIVIDE)
	default:
		return
	}
}

func (c *Compiler) number() {
	value, err := strconv.ParseFloat(c.previous.Lexeme(), 64)
	if err != nil {
		c.reportError(err.Error())
		return
	}
	c.emitByteCode(OP_CONST)
	c.makeConstant(value) // emits constant and the adds value to the value array
	// c.emitByteCode(OpCode(c.makeConstant(value)))
	// c.emitBytes(OP_CONST, OpCode(c.makeConstant(value)))
}

func (c *Compiler) makeConstant(value float64) uint8 {
	constant := c.chunk.WriteConstant(value, c.previous.Line)
	if constant > math.MaxInt {
		c.reportError("Too many constants in on chunk.")
		return 0
	}
	return uint8(constant)
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
