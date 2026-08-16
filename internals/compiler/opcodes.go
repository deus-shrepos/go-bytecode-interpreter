package compiler

//go:generate stringer -type=OpCode
type OpCode uint8

const (
	OP_RETURN OpCode = iota
	OP_CONST
	OP_NEGATE
	OP_ADD
	OP_SUBTRACT
	OP_MULTIPLY
	OP_DIVIDE
	OP_CONST_LONG
)
