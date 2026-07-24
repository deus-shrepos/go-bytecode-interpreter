package vm

import (
	"fmt"
	c "go-bytecode-interpter/internals/compiler"
	"go-bytecode-interpter/internals/debug"
	"go-bytecode-interpter/internals/memory"
	"go-bytecode-interpter/internals/value"
	"unsafe"
)

type InterpretResult uint8
type Value = value.Value

const (
	InterpretOk = iota
	InterpretCompilerError
	InterpretRuntimeError
)

type VM struct {
	ip    *uint8
	Chunk *c.Chunk
	Stack []Value
	trace bool
}

func NewVM(arena *memory.Arena, chunk *c.Chunk, trace bool) *VM {
	return &VM{
		Chunk: chunk,
		Stack: memory.AllocSliceCap[Value](arena, 0, 256),
		trace: trace,
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
			debug.DisassembleInstruction(*vm.Chunk, vm.offset())
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

func (vm *VM) Interpret(source string) {
	vm.ip = (*uint8)(unsafe.Pointer(&vm.Chunk.Code[0]))
	vm.Run()
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

func (vm *VM) Push(value Value) {
	vm.Stack = append(vm.Stack, value)
}

func (vm *VM) Pop() Value {
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
