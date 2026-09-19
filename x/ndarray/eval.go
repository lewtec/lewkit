package ndarray

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator runs a compiled kernel. CPU is the fallback; device backends
// register at higher weight via [github.com/lewtec/lewkit/x/driver/ndeval].
type Evaluator interface {
	Run(ctx context.Context, kernel *Kernel, output []float32) error
	Close() error
}

type cpuEvaluator struct{}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = cpuEvaluator{}

func (cpuEvaluator) Run(_ context.Context, kernel *Kernel, output []float32) error {
	if kernel == nil {
		return ErrOp
	}
	return kernel.EvalInto(output)
}

func (cpuEvaluator) Close() error { return nil }

func (cpuEvaluator) Name() string { return "cpu" }

// Open is the highest-weight compatible evaluator (Vulkan if a GPU
// driver registered, else CPU). Import
// [github.com/lewtec/lewkit/x/driver/prelude] or
// [github.com/lewtec/lewkit/x/driver/ndeval].
func Open(ctx context.Context) (Evaluator, error) {
	evaluator, err := driver.Get[Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndarray open", "evaluator", evaluatorName(evaluator))
	return evaluator, nil
}

func evaluatorName(evaluator Evaluator) string {
	if n, ok := evaluator.(interface{ Name() string }); ok {
		if name := n.Name(); name != "" {
			return name
		}
	}
	return fmt.Sprintf("%T", evaluator)
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
