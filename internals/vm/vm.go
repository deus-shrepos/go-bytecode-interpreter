package vm

import (
	"fmt"
	c "go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"unsafe"
)

type InterpretResult uint8

const (
	InterpretOk = iota
	InterpretCompilerError
	InterpretRuntimeError
)

type VM struct {
	ip    *uint8
	Chunk *c.Chunk
	Stack []float64
	trace bool

	arena *memory.Arena
}

func NewVM(arena *memory.Arena, trace bool) *VM {
	return &VM{
		Stack: memory.AllocSliceCap[float64](arena, 0, 256),
		trace: trace,
		arena: arena,
	}

}

func (vm *VM) Run() InterpretResult {
	for {
		if vm.trace {
			fmt.Printf("    ")
			for idx := 0; idx < len(vm.Stack); idx++ {
				fmt.Printf("[ ")
				fmt.Printf("%g", vm.Stack[idx])
				fmt.Printf(" ]")
			}
			fmt.Println()
			c.DisassembleInstruction(*vm.Chunk, vm.offset())
		}
		instruction := vm.readByte()
		switch c.OpCode(instruction) {
		case c.OP_RETURN:
			return InterpretOk
		case c.OP_CONST:
			constant := vm.Chunk.Consts[vm.readByte()]
			vm.Push(constant)
		case c.OP_NEGATE:
			vm.Push(-vm.Pop())
		case c.OP_ADD:
			a, b := popTwoNumbers(vm)
			vm.Push(a + b)
		case c.OP_DIVIDE:
			a, b := popTwoNumbers(vm)
			vm.Push(a / b)
		case c.OP_MULTIPLY:
			a, b := popTwoNumbers(vm)
			vm.Push(a * b)
		case c.OP_SUBTRACT:
			a, b := popTwoNumbers(vm)
			vm.Push(a - b)
		}
	}
}

func (vm *VM) Interpret(source []byte) InterpretResult {

	chunk := c.NewChunk(vm.arena)
	compiler := c.NewCompiler(vm.arena, &chunk, source)

	// compile the source and emit bytecode
	// and store that in the chunk
	if !compiler.Compile() {
		vm.arena.Free()
		return InterpretCompilerError
	}
	vm.Chunk = &chunk
	vm.ip = (*uint8)(unsafe.Pointer(&vm.Chunk.Code[0]))
	result := vm.Run()
	vm.arena.Free()
	return result
}

func (vm *VM) readByte() uint8 {
	b := *vm.ip
	vm.advance()
	return b
}

func (vm *VM) offset() int {
	base := unsafe.Pointer(&vm.Chunk.Code[0])
	return int(uintptr(unsafe.Pointer(vm.ip)) - uintptr(base))
}

func (vm *VM) Push(value float64) {
	vm.Stack = append(vm.Stack, value)
}

func (vm *VM) Pop() float64 {
	elem := vm.Stack[len(vm.Stack)-1]
	vm.Stack = vm.Stack[:len(vm.Stack)-1]
	return elem
}

func (vm *VM) resetStack() {
	vm.Stack = vm.Stack[:0]
}

func (vm *VM) advance() {
	vm.ip = (*uint8)(unsafe.Add(unsafe.Pointer(vm.ip), 1))
}
