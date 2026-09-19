package ndarray

import (
	"context"
	"fmt"
	"image"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
)

// Evaluator runs a compiled kernel. CPU is the fallback driver; device
// backends (Vulkan, …) register at higher weight.
type Evaluator interface {
	Run(ctx context.Context, kernel *Kernel, output []float32) error
	Close() error
}

type cpuEvaluator struct {
	scratch []float32
}

// CPU is the register-tape evaluator. Always available.
var CPU Evaluator = &cpuEvaluator{}

func (c *cpuEvaluator) Run(_ context.Context, kernel *Kernel, output []float32) error {
	if kernel == nil {
		return ErrOp
	}
	return kernel.EvalInto(output)
}

func (c *cpuEvaluator) RunRGBA(_ context.Context, kernel *Kernel, destination *image.RGBA) error {
	if kernel == nil || destination == nil {
		return ErrOp
	}
	size := kernel.size
	if destination.Rect.Dx()*destination.Rect.Dy()*4 != size {
		return fmt.Errorf("%w: image %d×%d×4 != %d", ErrSize, destination.Rect.Dx(), destination.Rect.Dy(), size)
	}
	if cap(c.scratch) < size {
		c.scratch = make([]float32, size)
	} else {
		c.scratch = c.scratch[:size]
	}
	if err := kernel.EvalInto(c.scratch); err != nil {
		return err
	}
	packRGBA(destination, c.scratch)
	return nil
}

func (c *cpuEvaluator) Close() error { return nil }

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
	evaluator, err := driver.Get[Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndarray open", "evaluator", evaluatorName(evaluator))
	return evaluator, nil
}

type imageEvaluator interface {
	RunRGBA(ctx context.Context, kernel *Kernel, destination *image.RGBA) error
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
	switch e := evaluator.(type) {
	case *cpuEvaluator:
		return "cpu"
	case *Vulkan:
		if e != nil && e.Device != nil {
			return "vulkan:" + e.Device.Name()
		}
		return "vulkan"
	default:
		return fmt.Sprintf("%T", evaluator)
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
