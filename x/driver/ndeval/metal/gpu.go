package metal

import (
	"context"
	"errors"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/internal/gpuhost"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/table"
	ffimetal "github.com/lewtec/lewkit/x/ffi/native/metal"
	"github.com/lewtec/lewkit/x/ndarray"
)

func init() {
	driver.Register[ndarray.Evaluator](factory{})
}

var _ driver.DriverFactory[ndarray.Evaluator] = factory{}
var _ ndarray.Evaluator = (*gpuEvaluator)(nil)
var _ ndarray.Program = (*session)(nil)

type factory struct{}

func (factory) ID() string   { return "ndeval_metal" }
func (factory) Name() string { return "Metal" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error { return gpuhost.Metal() }

func (factory) New(context.Context) (ndarray.Evaluator, error) {
	device, err := ffimetal.OpenDevice()
	if err != nil {
		return nil, err
	}
	return &gpuEvaluator{device: device, own: true}, nil
}

type gpuEvaluator struct {
	device *ffimetal.Device
	own    bool
	table  table.Table[*session]
}

func (g *gpuEvaluator) Name() string {
	if g == nil || g.device == nil {
		return "metal"
	}
	return "metal:" + g.device.Name()
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
		slog.Debug("ndeval metal session", "device", g.device.Name())
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
	var gpu *ffimetal.Device
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
		slog.Debug("ndeval metal close", "device", gpu.Name(), "sessions", len(sessions))
		err = errors.Join(err, gpu.Close())
	}
	return err
}
