package ndarray

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator binds a kernel to backend code (CPU tape, GPU session).
type Evaluator interface {
	Program(ctx context.Context, kernel *Kernel) (Program, error)
	Close() error
}

// Program is backend code for one tensor. Resize is visible on the next
// Eval. Close drops this binding; the evaluator may cache it.
type Program interface {
	Eval(ctx context.Context, output []byte) error
	Close() error
}

type cpuEvaluator struct {
	mu    sync.Mutex
	bound map[*Kernel]*cpuExec
}

type cpuExec struct {
	parent *cpuEvaluator
	kernel *Kernel
	tape   cpuProgram
}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = newCPUEvaluator()

func newCPUEvaluator() *cpuEvaluator {
	return &cpuEvaluator{bound: make(map[*Kernel]*cpuExec)}
}

func (c *cpuEvaluator) Program(_ context.Context, kernel *Kernel) (Program, error) {
	if c == nil || kernel == nil {
		return nil, ErrOp
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.bound[kernel]; ok {
		return e, nil
	}
	tape, err := lowerCPU(kernel.order, kernel.bufs, kernel.built)
	if err != nil {
		return nil, err
	}
	e := &cpuExec{parent: c, kernel: kernel, tape: tape}
	c.bound[kernel] = e
	return e, nil
}

func (e *cpuExec) Eval(_ context.Context, output []byte) error {
	if e == nil || e.kernel == nil {
		return ErrOp
	}
	k := e.kernel
	need := k.size * k.outType.size()
	if len(output) < need {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), need)
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
	if e.parent.bound[e.kernel] == e {
		delete(e.parent.bound, e.kernel)
	}
	e.parent.mu.Unlock()
	return nil
}

func (c *cpuEvaluator) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	c.bound = make(map[*Kernel]*cpuExec)
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
