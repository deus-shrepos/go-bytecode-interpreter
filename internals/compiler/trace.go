package compiler

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
)


type Trace struct {
	output io.Writer
	indent int
}

var depth atomic.Int32

func NewTrace(out io.Writer) Trace {
	if out == nil {
		out = os.Stdout
	}
	return Trace{output: out}
}

func setTrace(traceType uint8) func() {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return func(){}
	}
	funcName := runtime.FuncForPC(pc).Name()
	depth.Add(1)
	fmt.Printf("---> Entering %s \n"+strings.Repeat(" ", int(depth.Load())), funcName)
	return func() {
		fmt.Printf("<--- Exiting %s \n", funcName)
		depth.Add(-1)
	}
}
