package main

import (
	"flag"
	"fmt"
	"go-bytecode-interpreter/internals/compiler"
	"go-bytecode-interpreter/internals/source"
	"os"
	"strings"
)

const helpString = `
Valid Args:
 compile <path>: compile a glox program
 compile <path> --trace-compiler: load compile execution traces
 run <path>: execute a glox program		
`

func help() {
	fmt.Fprintf(os.Stderr, "%s", helpString)
	os.Exit(2)
}

func main() {
	if len(os.Args) < 2 {
		help()
	}

	switch os.Args[1] {
	case "compile":
		fs := flag.NewFlagSet("compile", flag.ExitOnError)
		traceCompiler := fs.Bool("trace-compiler", false, "trace compilation")
		tracePrecedence := fs.Bool("trace-prec", false, "trace tracedence")
		fs.Parse(os.Args[2:])
		if fs.NArg() == 0 {
			fmt.Fprintf(os.Stderr, "usage: glox compile <path> [--trace-compiler] [--trace-precedence]")
			os.Exit(2)
		}
		var trace uint8
		if *traceCompiler {
			fmt.Println("Trace Enabled")
			trace |= compiler.TraceCompiler
		}

		if *tracePrecedence {
			trace |= compiler.TracePrecedence
		}
		source.CompileProgram(strings.Trim(fs.Arg(0), " "), trace)
		os.Exit(0)
	case "run":
		fs := flag.NewFlagSet("run", flag.ExitOnError)
		if fs.NArg() != 1 {
			fmt.Fprintf(os.Stderr, "usage: glox run <path>")
		}
		source.RunProgram(fs.Arg(0))
		os.Exit(0)
	case "repl":
		fmt.Fprintf(os.Stdin, "Not implemented yet")
	default:
		help()
	}

}
