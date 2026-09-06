package repl

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

// stuff to work on.
func (r Repl) Cli() {
}
