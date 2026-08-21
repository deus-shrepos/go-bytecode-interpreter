package errors

import (
	"errors"
	"fmt"
	"go-bytecode-interpreter/internals/lexer"
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
	errorString.WriteString(fmt.Sprintf("[line %d] %s", e.token.Line, e.Phase))
	if e.token.Type == lexer.EOF {
		errorString.WriteString(" at end")
	} else if e.token.Type == lexer.ERROR {

	} else {
		errorString.WriteString(fmt.Sprintf(" at '%.*s'", e.token.Length, &e.token.Start))
	}
	return fmt.Sprintf(errorString.String())
}

func (e *Error) unwrap() error {
	return e.Phase
}

func (e *Error) errorAt() {

}
