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
		if os.Args[3] == "--compile-mode" {
			r.SetMode(repl.CompileMode)
		}
		if os.Args[3] == "--compile-debug-mode" {
			r.SetMode(repl.CompileDebugMode)
		}
		r.LoadProgramFromPath(os.Args[2])
	default:
		cliHelp()
	}

}
