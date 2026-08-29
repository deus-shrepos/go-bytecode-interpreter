package vm

func popTwoNumbers(vm *VM) (a, b float64) {
	b = vm.Pop()
	a = vm.Pop()
	return a, b
}
