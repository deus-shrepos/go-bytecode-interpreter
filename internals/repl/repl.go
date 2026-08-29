package repl

import (
	"fmt"
	"go-bytecode-interpreter/internals/memory"
	"go-bytecode-interpreter/internals/vm"
	"os"
)

type Repl struct{}

func NewCLI() *Repl {
	return &Repl{}
}

func (r Repl) LoadProgramFromPath(path string) {
	file, err := os.ReadFile(path)
	if err != nil {
		fmt.Errorf("cannot open the file: %v", err)
	}

	arena := memory.NewArena(1 << 16)
	vm := vm.NewVM(arena, true)
	vm.Interpret(file) // VM dispatch
}

// todo: we are going to do this later cuz we have other important
// stuff to work on.
func (r Repl) Cli() {
}
