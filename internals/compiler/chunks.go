package compiler

import (
	"go-bytecode-interpter/internals/memory"
	"go-bytecode-interpter/internals/value"
)

type Lines struct {
	Line  int
	Count int
}
type Chunk struct {
	Lines  []Lines
	Code   []OpCode
	Consts []value.Value
}

func NewChunk(arena *memory.Arena) Chunk {
	return Chunk{
		Lines:  memory.AllocSliceCap[Lines](arena, 0, 64),
		Code:   memory.AllocSliceCap[OpCode](arena, 0, 64),
		Consts: memory.AllocSliceCap[value.Value](arena, 0, 64),
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

func (c *Chunk) WriteConstant(value value.Value, line int) {
	idx := c.addConst(value)
	switch c.lastByteCode() {
	case OP_CONST:
		c.WriteChunk(OpCode(idx&0xff), line)
	case OP_CONST_LONG:
		c.WriteChunk(OpCode(idx&0xff), line)
		c.WriteChunk(OpCode((idx>>8)&0xff), line)
		c.WriteChunk(OpCode(idx>>16), line)
	}
}

func (c *Chunk) addConst(value value.Value) int {
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

func LoadLongConst(a, b, c uint8) uint8 {
	return (a & 0xff) | (b << 8) | (c << 16)
}
