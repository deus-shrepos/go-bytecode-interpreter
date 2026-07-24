package vm

func popTwoNumbers(vm *VM) (a, b Value) {
	b = vm.Pop()
	a = vm.Pop()
	return a, b
}
