package compiler

import (
	"fmt"
	"go-bytecode-interpreter/internals/errors"
	"go-bytecode-interpreter/internals/lexer"
	"go-bytecode-interpreter/internals/memory"
	"go-bytecode-interpreter/internals/token"
	"math"
	"os"
	"strconv"
)

const (
	TraceCompiler = 1 << iota
	TacePrecedence
)

type Compiler struct {
	current  token.Token
	previous token.Token
	scanner  lexer.Scanner

	chunk *Chunk
	arena *memory.Arena

	hadError  bool
	panicMode bool
	debugMode bool

	// Flag that traces entire compilation,
	// or just precedence
	Trace uint8

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

func (c *Compiler) SetTrace(trace uint8) {
	c.Trace = trace
}

func (c *Compiler) Compile() bool {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.advance()
	c.expression()
	c.consume(token.EOF, "Expect end of expression.")
	c.endCompiler()
	return !c.hadError
}

func (c *Compiler) advance() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.previous = c.current
	for {
		c.current = c.scanner.ScanToken()
		if c.current.Type != token.ERROR {
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
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.parsePrecedence(PrecAssignment)
}

func (c *Compiler) consume(tokeType token.Kind, message string) {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	if c.current.Type == tokeType {
		c.advance()
		return
	}
	c.reportError(message)
}

func (c *Compiler) emitByteCode(byteCode OpCode) {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	// write the chunk and record the previous line for runtime error info
	// if constant == -1 {
	// c.reportError("Expected CONST opcode.)}
	c.chunk.WriteChunk(byteCode, c.previous.Line)
	if constant == -1 {
		c.reportError("Expected CONST opcode.")
	}
}

func (c *Compiler) endCompiler() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.emitByteCode(OP_RETURN)
	if c.debugMode && c.hadError {
		DisassembleChunk(*c.chunk, "code")
	}
}

func (c *Compiler) emitReturn() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.emitByteCode(OP_RETURN)
}

func (c *Compiler) emitBytes(b1 OpCode, b2 OpCode) {
	c.emitByteCode(b1)
	c.emitByteCode(b2)
}

func (c *Compiler) grouping() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	c.expression()
	c.consume(token.RIGHT_PAREN, "Expect ')' after expression.")
}

func (c *Compiler) parsePrecedence(prec Precedence) {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
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
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	operaterType := c.previous.Type

	// compile the operand
	c.parsePrecedence(PrecUnary)
	switch operaterType {
	case token.MINUS:
		c.emitByteCode(OP_NEGATE)
	default:
		return
	}

}

func (c *Compiler) binary() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	operatorType := c.previous.Type

	rule := rules[operatorType]
	// left-assoc precedence
	c.parsePrecedence(Precedence(rule.prec + 1))

	switch operatorType {
	case token.PLUS:
		c.emitByteCode(OP_ADD)
	case token.MINUS:
		c.emitByteCode(OP_SUBTRACT)
	case token.STAR:
		c.emitByteCode(OP_MULTIPLY)
	case token.SLASH:
		c.emitByteCode(OP_DIVIDE)
	default:
		return
	}
}

func (c *Compiler) number() {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	value, err := strconv.ParseFloat(c.previous.Lexeme(), 64)
	if err != nil {
		c.reportError(err.Error())
		return
	}
	c.emitByteCode(OP_CONST)
	c.makeConstant(value) // emits constant and the adds value to the value array
}

func (c *Compiler) makeConstant(value float64) uint8 {
	if c.isTraceEnabled() {
		defer setTrace(c.Trace)()
	}
	constant := c.chunk.WriteConstant(value, c.previous.Line)
	if constant == -1 {
		c.reportError("Expected CONST opcode.")
	}
	if constant > math.MaxInt {
		c.reportError("Too many constants in on chunk.")
		return 0
	}
	return uint8(constant)
}

func (c *Compiler) reportError(message string) {
	c.panicMode = true
	err := errors.NewError(c.current, errors.CompilePhaseError, message)
	fmt.Fprintf(os.Stderr, "%s", err.Error())
	c.hadError = true
}

func (c *Compiler) isTraceEnabled() bool {
	return c.Trace&0xff == 1
}
