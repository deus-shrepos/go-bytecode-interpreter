package debug

import (
	"fmt"
	"go-bytecode-interpreter/internals/compiler"
	_ "go-bytecode-interpreter/internals/compiler"
)

func simpleInstr(opcode compiler.OpCode, offset int) int {
	fmt.Println(opcode)
	return offset + 1
}

func constantInstr(opcode compiler.OpCode, chunk compiler.Chunk, offset int) int {
	constant := chunk.Code[offset+1]
	fmt.Printf("%-16s %4d '", opcode, constant)
	fmt.Printf("%g'", chunk.Consts[constant])
	fmt.Println()
	return offset + 2

}

func loadConstInstr(opcode compiler.OpCode, chunk compiler.Chunk, offset int) int {
	constant := compiler.LoadLongConst(uint8(chunk.Code[offset+1]), uint8(chunk.Code[offset+2]), uint8(chunk.Code[offset+3]))
	fmt.Printf("%-16s %4d ", opcode, constant)
	fmt.Printf("%g", chunk.Consts[constant])
	fmt.Println()
	return offset + 2
}

func DisassembleChunk(chunk compiler.Chunk, name string) {
	fmt.Printf("=== %s === \n", name)
	for offset := 0; offset < len(chunk.Code); {
		offset = DisassembleInstruction(chunk, offset)
	}
}

func DisassembleInstruction(chunk compiler.Chunk, offset int) int {
	fmt.Printf("%04d ", offset)

	isNewLine := false
	if compiler.GetLine(chunk, offset) == compiler.GetLine(chunk, offset-1) {
		isNewLine = true
	}

	if offset > 0 && isNewLine {
		fmt.Printf("   | ")
	} else {
		fmt.Printf("%4d ", compiler.GetLine(chunk, offset))
	}

	instruction := chunk.Code[offset]
	switch instruction {
	case compiler.OP_RETURN:
		return simpleInstr(instruction, offset)
	case compiler.OP_NEGATE:
		return simpleInstr(instruction, offset)
	case compiler.OP_ADD:
		return simpleInstr(instruction, offset)
	case compiler.OP_DIVIDE:
		return simpleInstr(instruction, offset)
	case compiler.OP_MULTIPLY:
		return simpleInstr(instruction, offset)
	case compiler.OP_SUBTRACT:
		return simpleInstr(instruction, offset)
	case compiler.OP_CONST:
		return constantInstr(instruction, chunk, offset)
	case compiler.OP_CONST_LONG:
		return loadConstInstr(instruction, chunk, offset)
	default:
		fmt.Printf("Unknown opcode %v\n", instruction)
		return offset + 1
	}
}
