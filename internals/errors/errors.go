package errors

import (
	"errors"
	"fmt"
	"go-bytecode-interpreter/internals/lexer"
	"os"
	"strings"
)

var lexerPhaseError = errors.New("LexerPhase")
var CompilePhaseError = errors.New("CompilerPhase")
var RuntimePhaseError = errors.New("RuntimePhase")

type Error struct {
	Phase   error
	Message string
	token   lexer.Token
}

func NewError(token lexer.Token, phase error, msg string) Error {

	return Error{
		Phase:   phase,
		Message: msg,
		token:   token,
	}
}

func (e *Error) Error() string {
	errorString := strings.Builder{}
	fmt.Fprintf(&errorString, "[line %d] %s", e.token.Line, e.Phase)
	switch e.token.Type {
	case lexer.EOF:
		fmt.Fprintf(&errorString, " at end")
	case lexer.ERROR:
		break
	default:
		fmt.Fprintf(&errorString, " at '%s'", e.token.Lexeme())
	}

	fmt.Fprintf(os.Stderr, ": %s\n", e.Message)
	return fmt.Sprintf(errorString.String())
}

func (e *Error) Unwrap() error {
	return e.Phase
}
