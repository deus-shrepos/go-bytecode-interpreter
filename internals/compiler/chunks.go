package compiler

import (
	"go-bytecode-interpreter/internals/memory"
)

type Lines struct {
	Line  int
	Count int
}
type Chunk struct {
	Lines  []Lines
	Code   []OpCode
	Consts []float64
}

func NewChunk(arena *memory.Arena) Chunk {
	return Chunk{
		Lines:  memory.AllocSliceCap[Lines](arena, 0, 64),
		Code:   memory.AllocSliceCap[OpCode](arena, 0, 64),
		Consts: memory.AllocSliceCap[float64](arena, 0, 64),
	}
}

func (c *Chunk) WriteChunk(opCode OpCode, line int) {
	c.Code = append(
		c.Code,
		opCode,
	)

	// Run-length encoding stuff
	if len(c.Code) == 1 {
		c.Lines = append(c.Lines, Lines{Line: line, Count: 1})
		return
	}

	if c.Lines[len(c.Lines)-1].Line == line {
		c.Lines[len(c.Lines)-1].Count++
	} else {
		c.Lines = append(c.Lines, Lines{Line: line, Count: 1})
	}
}

func (c *Chunk) WriteConstant(value float64, line int) int {
	idx := c.AddConstant(value)

	//nolint:exhaustive
	switch c.lastByteCode() {
	case OP_CONST:
		c.WriteChunk(OpCode(idx&0xff), line)
	case OP_CONST_LONG:
		c.WriteChunk(OpCode(idx&0xff), line)
		c.WriteChunk(OpCode((idx>>8)&0xff), line)
		c.WriteChunk(OpCode(idx>>16), line)
	default:
		return -1 // incorrect opcode
	}
	return idx
}

func (c *Chunk) AddConstant(value float64) int {
	c.Consts = append(c.Consts, value)
	return len(c.Consts) - 1 // return the index of the constant
}

func (c *Chunk) lastByteCode() OpCode {
	return c.Code[len(c.Code)-1]
}

func GetLine(chunk Chunk, offset int) int {
	idx := 0
	for idx < len(chunk.Lines)-1 && chunk.Lines[idx].Count <= offset {
		offset -= chunk.Lines[idx].Count
		idx++
	}

	return chunk.Lines[idx].Line
}

func LoadLongConst(a, b, c uint8) uint32 {
	return (uint32(a) & 0xff) | (uint32(b) << 8) | (uint32(c) << 16)
}
