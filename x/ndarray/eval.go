package ndarray

import (
	"context"
	"fmt"
	"image"
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

func packRGBA(destination *image.RGBA, source []float32) {
	if destination == nil {
		return
	}
	width, height := destination.Rect.Dx(), destination.Rect.Dy()
	if width < 1 || height < 1 || len(source) < height*width*4 {
		return
	}
	for y := range height {
		destIndex := destination.PixOffset(destination.Rect.Min.X, destination.Rect.Min.Y+y)
		sourceIndex := y * width * 4
		for range width {
			destination.Pix[destIndex] = toUint8(source[sourceIndex])
			destination.Pix[destIndex+1] = toUint8(source[sourceIndex+1])
			destination.Pix[destIndex+2] = toUint8(source[sourceIndex+2])
			destination.Pix[destIndex+3] = toUint8(source[sourceIndex+3])
			destIndex += 4
			sourceIndex += 4
		}
	}
}

func toUint8(v float32) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
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
