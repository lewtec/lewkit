package ndarray

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator runs a compiled kernel. CPU is the fallback driver; device
// backends (Vulkan, …) register at higher weight.
type Evaluator interface {
	Run(ctx context.Context, k *Kernel, output []float32) error
	Close() error
}

type cpu struct{}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = cpu{}

func (cpu) Run(_ context.Context, k *Kernel, output []float32) error {
	if k == nil {
		return ErrOp
	}
	return k.EvalInto(output)
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
	ev, err := driver.Get[Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndarray open", "evaluator", evaluatorName(ev))
	return ev, nil
}

func evaluatorName(ev Evaluator) string {
	switch e := ev.(type) {
	case cpu:
		return "cpu"
	case *Vulkan:
		if e != nil && e.Device != nil {
			return "vulkan:" + e.Device.Name()
		}
		return "vulkan"
	default:
		return fmt.Sprintf("%T", ev)
	}
}

// Eval runs the kernel on the CPU. inputs[i] is the buffer for each source.
// It interprets a register tape built at compile (no native codegen) and
// shards cells across GOMAXPROCS when the output is large enough.
func (k *Kernel) Eval() ([]float32, error) {
	if k == nil || k.cpu.registers == 0 {
		return nil, ErrOp
	}
	if k.size == 0 {
		return nil, nil
	}
	output := make([]float32, k.size)
	if err := k.EvalInto(output); err != nil {
		return nil, err
	}
	return output, nil
}

// EvalInto writes the kernel into output, which must have length at least size.
// Inputs are k.bufs, each a contiguous []float32.
func (k *Kernel) EvalInto(output []float32) error {
	if k == nil || k.cpu.registers == 0 {
		return ErrOp
	}
	if len(output) < k.size {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), k.size)
	}
	if k.size == 0 {
		return nil
	}
	k.evalCPU(output)
	return nil
}
