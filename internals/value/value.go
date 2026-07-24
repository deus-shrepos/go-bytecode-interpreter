package value

import (
	"fmt"
	mem "go-bytecode-interpter/internals/memory"
)

type Value float64

type ValueArray struct {
	Count  int
	Values []Value
}

func NewValueArray() ValueArray {
	a := mem.NewArena(8)
	return ValueArray{
		Count:  0,
		Values: mem.AllocSlice[Value](a, 8),
	}
}

func (v *ValueArray) WriteValueArray(value Value) {
	v.Values = append(v.Values, value)
	v.Count += 1
}

func (v *ValueArray) PrintValue(value Value) {
	fmt.Printf("%g", value)
}
