package compiler

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"
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

func setTrace(trace uint8) func() {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return func() {}
	}
	funcName := filterFuncName(runtime.FuncForPC(pc).Name())
	start := time.Now()
	d := depth.Add(1) - 1
	fmt.Fprintf(os.Stdin, "%s --> Entering %s() \n", strings.Repeat(" ", int(d)), funcName)
	return func() {
		fmt.Fprintf(os.Stdin, "%s <-- Exiting %s() (took %s)\n", strings.Repeat(" ", int(d)), funcName, time.Since(start))
		depth.Add(-1)
	}

}

func filterFuncName(name string) string {
	str1 := strings.SplitAfterN(name, ".", 2)[1]
	return strings.SplitN(str1, ".", 2)[1]
}
