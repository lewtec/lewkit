package vulkan

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/table"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

func init() {
	driver.Register[ndarray.Evaluator](gpuFactory{})
}

var _ driver.DriverFactory[ndarray.Evaluator] = gpuFactory{}
var _ ndarray.Evaluator = (*gpuEvaluator)(nil)
var _ ndarray.Program = (*session)(nil)

type gpuFactory struct{}

func (gpuFactory) ID() string   { return "ndeval_vulkan" }
func (gpuFactory) Name() string { return "Vulkan" }
func (gpuFactory) Weight() int  { return 50 }

func (gpuFactory) CheckCompatibility(ctx context.Context) error {
	_, err := drvvulkan.List(ctx)
	return err
}

func (gpuFactory) New(ctx context.Context) (ndarray.Evaluator, error) {
	gpu, err := drvvulkan.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &gpuEvaluator{device: gpu, own: true}, nil
}

type gpuEvaluator struct {
	device      drvvulkan.Device
	own         bool
	table       table.Table[*session]
	present     []byte
	presentErr  error
	presentDone bool
}

func (g *gpuEvaluator) Name() string {
	if g == nil || g.device == nil {
		return "vulkan"
	}
	return "vulkan:" + g.device.Name()
}

func (g *gpuEvaluator) Program(ctx context.Context, kernel *ndarray.Kernel) (ndarray.Program, error) {
	if kernel == nil {
		return nil, ndarray.ErrOp
	}
	return g.session(ctx, kernel)
}

func (g *gpuEvaluator) session(ctx context.Context, kernel *ndarray.Kernel) (*session, error) {
	if g.device == nil || kernel == nil {
		return nil, ndarray.ErrOp
	}
	s, created, err := g.table.Open(kernel, sameKernel(kernel), func() (*session, error) {
		return newSession(ctx, kernel, g.device)
	}, func(s *session) {
		s.eval = &g.table.Eval
		s.forget = g.table.Forget(kernel, s)
	})
	if err != nil {
		return nil, err
	}
	if created {
		slog.Debug("ndeval vulkan session", "device", g.device.Name())
	}
	return s, nil
}

func sameKernel(kernel *ndarray.Kernel) func(*session) bool {
	return func(s *session) bool { return s.kernel == kernel }
}

func (g *gpuEvaluator) Close() error {
	if g == nil {
		return nil
	}
	var own bool
	var gpu drvvulkan.Device
	sessions := g.table.Take(func() {
		own, gpu = g.own, g.device
		g.own = false
		g.device = nil
	})
	var err error
	for _, s := range sessions {
		err = errors.Join(err, s.Close())
	}
	if own && gpu != nil {
		slog.Debug("ndeval vulkan close", "device", gpu.Name(), "sessions", len(sessions))
		err = errors.Join(err, gpu.Close())
	}
	return err
}
