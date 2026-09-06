package errors

import (
	"errors"
	"fmt"
	"go-bytecode-interpreter/internals/token"
	"os"
	"strings"
)

var lexerPhaseError = errors.New("LexerError")
var CompilePhaseError = errors.New("CompileError")
var RuntimePhaseError = errors.New("RuntimeError")

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
	fmt.Fprintf(&err, "%s:[line %d]", e.Phase, e.token.Line)
	switch e.token.Type {
	case token.EOF:
		fmt.Fprintf(&err, " at end")
	case token.ERROR:
		break
	default:
		fmt.Fprintf(&err, " at '%s'", e.token.Lexeme())
	}
	fmt.Fprintf(os.Stderr, ":%s\n", e.Message)
	return err.String()
}

func (e Error) Unwrap() error {
	return e.Phase
}
