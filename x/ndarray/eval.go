package ndarray

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator binds a tensor to backend code (CPU tape, GPU session).
type Evaluator interface {
	Program(ctx context.Context, tensor *Tensor) (Program, error)
	Close() error
}

// Program is backend code for one tensor. Resize is visible on the next
// Eval. Close drops this binding; the evaluator may cache it.
type Program interface {
	Eval(ctx context.Context, output []float32) error
	Close() error
}

type cpuEvaluator struct {
	mu    sync.Mutex
	bound map[*Tensor]*cpuExec
}

type cpuExec struct {
	parent *cpuEvaluator
	tensor *Tensor
	kernel *Kernel
	tape   cpuProgram
}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = newCPUEvaluator()

func newCPUEvaluator() *cpuEvaluator {
	return &cpuEvaluator{bound: make(map[*Tensor]*cpuExec)}
}

func (c *cpuEvaluator) Program(_ context.Context, tensor *Tensor) (Program, error) {
	if c == nil || tensor == nil || tensor.kernel == nil {
		return nil, ErrOp
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.bound[tensor]; ok && e.kernel == tensor.kernel {
		return e, nil
	}
	if old := c.bound[tensor]; old != nil {
		delete(c.bound, tensor)
	}
	tape, err := lowerCPU(tensor.kernel.order, tensor.kernel.bufs, tensor.kernel.built)
	if err != nil {
		return nil, err
	}
	e := &cpuExec{parent: c, tensor: tensor, kernel: tensor.kernel, tape: tape}
	c.bound[tensor] = e
	return e, nil
}

func (e *cpuExec) Eval(_ context.Context, output []float32) error {
	if e == nil || e.tensor == nil || e.tensor.kernel == nil {
		return ErrOp
	}
	k := e.tensor.kernel
	if len(output) < k.size {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), k.size)
	}
	if k.size == 0 {
		return nil
	}
	runCPU(e.tape, k, output)
	return nil
}

func (e *cpuExec) Close() error {
	if e == nil || e.parent == nil {
		return nil
	}
	e.parent.mu.Lock()
	if e.parent.bound[e.tensor] == e {
		delete(e.parent.bound, e.tensor)
	}
	e.parent.mu.Unlock()
	return nil
}

func (c *cpuEvaluator) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.bound = make(map[*Tensor]*cpuExec)
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
