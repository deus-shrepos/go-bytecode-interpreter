package compiler

import (
	"fmt"
)

func simpleInstr(opcode OpCode, offset int) int {
	fmt.Println(opcode)
	return offset + 1
}

func constantInstr(opcode OpCode, chunk Chunk, offset int) int {
	constant := chunk.Code[offset+1]
	fmt.Printf("%-16s %4d '", opcode, constant)
	fmt.Printf("%g'", chunk.Consts[constant])
	fmt.Println()
	return offset + 2

}

func loadConstInstr(opcode OpCode, chunk Chunk, offset int) int {
	constant := LoadLongConst(uint8(chunk.Code[offset+1]), uint8(chunk.Code[offset+2]), uint8(chunk.Code[offset+3]))
	fmt.Printf("%-16s %4d ", opcode, constant)
	fmt.Printf("%g", chunk.Consts[constant])
	fmt.Println()
	return offset + 2
}

func DisassembleChunk(chunk Chunk, name string) {
	fmt.Printf("=== %s === \n", name)
	for offset := 0; offset < len(chunk.Code); {
		offset = DisassembleInstruction(chunk, offset)
	}
}

func DisassembleInstruction(chunk Chunk, offset int) int {
	fmt.Printf("%04d ", offset)

	isNewLine := false
	if GetLine(chunk, offset) == GetLine(chunk, offset-1) {
		isNewLine = true
	}

	if offset > 0 && isNewLine {
		fmt.Printf("   | ")
	} else {
		fmt.Printf("%4d ", GetLine(chunk, offset))
	}

	instruction := chunk.Code[offset]
	switch instruction {
	case OP_RETURN:
		return simpleInstr(instruction, offset)
	case OP_NEGATE:
		return simpleInstr(instruction, offset)
	case OP_ADD:
		return simpleInstr(instruction, offset)
	case OP_DIVIDE:
		return simpleInstr(instruction, offset)
	case OP_MULTIPLY:
		return simpleInstr(instruction, offset)
	case OP_SUBTRACT:
		return simpleInstr(instruction, offset)
	case OP_CONST:
		return constantInstr(instruction, chunk, offset)
	case OP_CONST_LONG:
		return loadConstInstr(instruction, chunk, offset)
	default:
		fmt.Printf("Unknown opcode %v\n", instruction)
		return offset + 1
	}
}
