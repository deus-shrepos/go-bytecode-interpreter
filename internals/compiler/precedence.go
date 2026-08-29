package compiler

import (
	"go-bytecode-interpreter/internals/lexer"

	"honnef.co/go/tools/analysis/facts/tokenfile"
)

type Precedence uint8

const (
	PrecNone Precedence = iota
	PrecAssignment
	PrecOr
	PrecAnd
	PrecEquality
	PrecComparison
	PrecTerm
	PrecFactor
	PrecUnary
	PrecCall
	PrecPrimary
)

type ParseFunc *func()

type ParseRule struct {
	prefix ParseFunc
	infix  ParseFunc
	prec   Precedence
}

var rules = map[lexer.TokenType]ParseRule{
	lexer.LEFT_BRACE: ParseRule{grouping, nil, PREC_NONE}
}
