package ndarray

import "fmt"

// Eval runs the kernel on the CPU. srcs[i] is the buffer for Slots()[i].
// It interprets a register tape built at Compile (no native codegen) and
// shards cells across GOMAXPROCS when the output is large enough.
func (k *Kernel) Eval(srcs ...[]float32) ([]float32, error) {
	if k == nil || k.cpu.nreg == 0 {
		return nil, ErrOp
	}
	if len(srcs) != len(k.slots) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(srcs))
	}
	if k.n == 0 {
		return nil, nil
	}
	dst := make([]float32, k.n)
	if err := k.EvalInto(dst, srcs); err != nil {
		return nil, err
	}
	return dst, nil
}

// EvalInto writes the kernel into dst, which must have length at least Size.
func (k *Kernel) EvalInto(dst []float32, srcs [][]float32) error {
	if k == nil || k.cpu.nreg == 0 {
		return ErrOp
	}
	if len(srcs) != len(k.slots) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(srcs))
	}
	if len(dst) < k.n {
		return fmt.Errorf("%w: dst %d < %d", ErrSize, len(dst), k.n)
	}
	if k.n == 0 {
		return nil
	}
	k.evalCPU(dst, srcs)
	return nil
}
