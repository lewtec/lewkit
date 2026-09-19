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
	return k.evalCPU(srcs), nil
}
