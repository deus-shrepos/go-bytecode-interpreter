package repl

import (
	"fmt"
	"go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/memory"
	"go-bytecode-interpreter/internals/vm"
	"os"
)

type ReplMode uint8

const (
	NormalMode = 1 << iota
	CompileMode
	CompileDebugMode
	VMDebugMode
)

type Repl struct {
	Mode ReplMode
}

func NewCLI() *Repl {
	return &Repl{}
}

func (r *Repl) SetMode(mode ReplMode) {
	r.Mode = mode
}

func (r *Repl) GetMode() ReplMode {
	return r.Mode
}

func (r *Repl) LoadProgramFromPath(path string) {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Errorf("cannot open the file: %v", err)
	}
	arena := memory.NewArena(256)

	mode := r.GetMode()

	switch mode {
	case CompileMode:
	case CompileDebugMode:
		chunk := compiler.NewChunk(arena)
		c := compiler.NewCompiler(arena, &chunk, file)
		if mode == CompileDebugMode {
			c.SetTrace(compiler.TraceCompiler)
		}
		c.Compile()

	case NormalMode:
		vm := vm.NewVM(arena, true)
		vm.Interpret(file) // VM dispatch
	}
}

// todo: we are going to do this later cuz we have other important
// stuff to work on.
func (r Repl) Cli() {
}
