package errors

import (
	"errors"
	"fmt"
	"go-bytecode-interpreter/internals/token"
	"os"
	"strings"
)

var lexerPhaseError = errors.New("LexerPhase")
var CompilePhaseError = errors.New("CompilerPhase")
var RuntimePhaseError = errors.New("RuntimePhase")

type Error struct {
	Phase   error
	Message string
	token   token.Token
}

func NewError(token token.Token, phase error, msg string) Error {

	return Error{
		Phase:   phase,
		Message: msg,
		token:   token,
	}
}

func (e Error) Error() string {
	err := strings.Builder{}
	fmt.Fprintf(&err, "[line %d] %s", e.token.Line, e.Phase)
	switch e.token.Type {
	case token.EOF:
		fmt.Fprintf(&err, " at end")
	case token.ERROR:
		break
	default:
		fmt.Fprintf(&err, " at '%s'", e.token.Lexeme())
	}
	fmt.Fprintf(os.Stderr, ": %s\n", e.Message)
	return err.String()
}

func (e Error) Unwrap() error {
	return e.Phase
}
