package ndarray

import "fmt"

// Eval runs the kernel on the CPU. srcs[i] is the buffer for Slots()[i].
// It interprets a register tape built at Compile; it does not emit or load
// native code.
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
	return k.cpu.eval(k.n, k.shape, k.outDT, srcs), nil
}
