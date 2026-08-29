package compiler

import (
	token "go-bytecode-interpreter/internals/lexer"
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

type ParseFunc func(*Compiler)

type ParseRule struct {
	prefix ParseFunc
	infix  ParseFunc
	prec   Precedence
}

var rules map[token.TokenType]ParseRule

func initRules() bool {
	rules = map[token.TokenType]ParseRule{
		token.LEFT_PAREN:    {(*Compiler).grouping, nil, PrecNone},
		token.RIGHT_PAREN:   {nil, nil, PrecNone},
		token.LEFT_BRACE:    {nil, nil, PrecNone},
		token.RIGHT_BRACE:   {nil, nil, PrecNone},
		token.COMMA:         {nil, nil, PrecNone},
		token.DOT:           {nil, nil, PrecNone},
		token.MINUS:         {(*Compiler).unary, (*Compiler).binary, PrecTerm},
		token.PLUS:          {nil, (*Compiler).binary, PrecTerm},
		token.SEMICOLON:     {nil, nil, PrecNone},
		token.SLASH:         {nil, (*Compiler).binary, PrecFactor},
		token.STAR:          {nil, (*Compiler).binary, PrecFactor},
		token.BANG:          {nil, nil, PrecNone},
		token.BANG_EQUAL:    {nil, nil, PrecNone},
		token.EQUAL:         {nil, nil, PrecNone},
		token.EQUAL_EQUAL:   {nil, nil, PrecNone},
		token.GREATER:       {nil, nil, PrecNone},
		token.GREATER_EQUAL: {nil, nil, PrecNone},
		token.NUMBER:        {(*Compiler).number, nil, PrecNone},
		token.PRINT:         {nil, nil, PrecNone},
	}

	// random number
	return true
}

var _ = initRules()
