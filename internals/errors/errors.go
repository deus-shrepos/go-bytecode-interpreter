package errors

import (
	"errors"
	"fmt"
)

var lexerPhaseError = errors.New("LexerPhase")
var CompilePhaseError = errors.New("CompilerPhase")
var RuntimePhaseError = errors.New("RuntimePhase")

type InterpreterError struct {
	Phase   error
	Line    int
	Message string
}

func (e *InterpreterError) Error() string {
	return fmt.Sprintf("%s at %d: %s", e.Phase, e.Line, e.Message)
}

func (e *InterpreterError) unwrap() error {
	return e.Phase
}
