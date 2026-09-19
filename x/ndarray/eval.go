package ndarray

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator runs a compiled kernel. CPU is the fallback driver; device
// backends (Vulkan, …) register at higher weight.
type Evaluator interface {
	Run(ctx context.Context, k *Kernel, output []float32, inputs [][]float32) error
	Close() error
}

type cpu struct{}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = cpu{}

func (cpu) Run(_ context.Context, k *Kernel, output []float32, inputs [][]float32) error {
	if k == nil {
		return ErrOp
	}
	return k.EvalInto(output, inputs)
}

func (cpu) Close() error { return nil }

type cpuFactory struct{}

func (cpuFactory) ID() string                               { return "ndarray_cpu" }
func (cpuFactory) Name() string                             { return "CPU" }
func (cpuFactory) Weight() int                              { return 0 }
func (cpuFactory) CheckCompatibility(context.Context) error { return nil }
func (cpuFactory) New(context.Context) (Evaluator, error)   { return CPU, nil }

func init() {
	driver.Register[Evaluator](cpuFactory{})
}

// Open is the highest-weight compatible evaluator (Vulkan if it can open, else CPU).
func Open(ctx context.Context) (Evaluator, error) {
	return driver.Get[Evaluator](ctx)
}

// Eval runs the kernel on the CPU. inputs[i] is the buffer for each source.
// It interprets a register tape built at compile (no native codegen) and
// shards cells across GOMAXPROCS when the output is large enough.
func (k *Kernel) Eval(inputs ...[]float32) ([]float32, error) {
	if k == nil || k.cpu.registers == 0 {
		return nil, ErrOp
	}
	if len(inputs) != len(k.bufs) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.bufs), len(inputs))
	}
	if k.size == 0 {
		return nil, nil
	}
	output := make([]float32, k.size)
	if err := k.EvalInto(output, inputs); err != nil {
		return nil, err
	}
	return output, nil
}

// EvalInto writes the kernel into output, which must have length at least size.
func (k *Kernel) EvalInto(output []float32, inputs [][]float32) error {
	if k == nil || k.cpu.registers == 0 {
		return ErrOp
	}
	if len(inputs) != len(k.bufs) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.bufs), len(inputs))
	}
	if len(output) < k.size {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), k.size)
	}
	if k.size == 0 {
		return nil
	}
	k.evalCPU(output, inputs)
	return nil
}
