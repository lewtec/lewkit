package ndarray

import "fmt"

// Eval runs the kernel on the CPU. inputs[i] is the buffer for Slots()[i].
// It interprets a register tape built at Compile (no native codegen) and
// shards cells across GOMAXPROCS when the output is large enough.
func (k *Kernel) Eval(inputs ...[]float32) ([]float32, error) {
	if k == nil || k.cpu.registers == 0 {
		return nil, ErrOp
	}
	if len(inputs) != len(k.slots) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(inputs))
	}
	if k.n == 0 {
		return nil, nil
	}
	output := make([]float32, k.n)
	if err := k.EvalInto(output, inputs); err != nil {
		return nil, err
	}
	return output, nil
}

// EvalInto writes the kernel into output, which must have length at least n.
func (k *Kernel) EvalInto(output []float32, inputs [][]float32) error {
	if k == nil || k.cpu.registers == 0 {
		return ErrOp
	}
	if len(inputs) != len(k.slots) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(inputs))
	}
	if len(output) < k.n {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), k.n)
	}
	if k.n == 0 {
		return nil
	}
	k.evalCPU(output, inputs)
	return nil
}
