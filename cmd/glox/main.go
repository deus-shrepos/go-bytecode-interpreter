package main

import (
	"fmt"
	"go-bytecode-interpreter/internals/repl"
	"os"
)

func cliHelp() {
	fmt.Println("Valid Args:")
	fmt.Println("--path: load from a specific path")
	fmt.Println("--cli:  interactive repl")
}

func main() {
	r := repl.NewCLI()
	switch os.Args[1] {
	case "--interactive":
		r.Cli()
	case "--path":
		r.LoadProgramFromPath(os.Args[2])
	default:
		cliHelp()
	}

}
