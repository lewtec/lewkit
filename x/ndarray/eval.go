package ndarray

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator runs a tensor. CPU is the fallback; device backends
// register at higher weight via [github.com/lewtec/lewkit/x/driver/ndeval].
type Evaluator interface {
	Run(ctx context.Context, tensor *Tensor, output []float32) error
	Close() error
}

type cpuEvaluator struct {
	mu    sync.Mutex
	tapes map[*Kernel]cpuProgram
}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = newCPUEvaluator()

func newCPUEvaluator() *cpuEvaluator {
	return &cpuEvaluator{tapes: make(map[*Kernel]cpuProgram)}
}

func (c *cpuEvaluator) Run(_ context.Context, tensor *Tensor, output []float32) error {
	if c == nil || tensor == nil || tensor.kernel == nil {
		return ErrOp
	}
	program, err := c.program(tensor.kernel)
	if err != nil {
		return err
	}
	if len(output) < tensor.kernel.size {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), tensor.kernel.size)
	}
	if tensor.kernel.size == 0 {
		return nil
	}
	runCPU(program, tensor.kernel, output)
	return nil
}

func (c *cpuEvaluator) program(k *Kernel) (cpuProgram, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p, ok := c.tapes[k]; ok {
		return p, nil
	}
	p, err := lowerCPU(k.order, k.bufs, k.built)
	if err != nil {
		return cpuProgram{}, err
	}
	c.tapes[k] = p
	return p, nil
}

func (c *cpuEvaluator) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.tapes = make(map[*Kernel]cpuProgram)
	c.mu.Unlock()
	return nil
}

func (*cpuEvaluator) Name() string { return "cpu" }

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
