package compiler

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

func (o OpCode) String() string {
	switch o {
	case OP_ADD:
		return "OP_ADD"
	case OP_CONST:
		return "OP_CONST"
	case OP_NEGATE:
		return "OP_NEGATE"
	case OP_SUBTRACT:
		return "OP_SUBTRACT"
	case OP_DIVIDE:
		return "OP_DIVIDE"
	case OP_CONST_LONG:
		return "OP_CONST_LONG"
	case OP_MULTIPLY:
		return "OP_MULTIPLY"
	case OP_RETURN:
		return "OP_RETURN"
	default:
		return "UNKNOW OP_CODE"
	}

}
