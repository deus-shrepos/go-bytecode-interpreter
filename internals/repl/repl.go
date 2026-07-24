package repl

import (
	"fmt"
	"go-bytecode-interpter/internals/compiler"
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

	compiler := compiler.NewCompiler()
	compiler.Compile(file)
}

// todo: we are going to do this later cuz we have other important
// stuff to work on.
func (r Repl) Cli() {
}
