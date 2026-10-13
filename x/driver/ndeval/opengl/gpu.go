package opengl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/table"
	ffiopengl "github.com/lewtec/lewkit/x/ffi/native/opengl"
	"github.com/lewtec/lewkit/x/ndarray"
)

func init() {
	driver.Register[ndarray.Evaluator](factory{})
}

var _ driver.DriverFactory[ndarray.Evaluator] = factory{}
var _ ndarray.Evaluator = (*gpuEvaluator)(nil)
var _ ndarray.Program = (*session)(nil)

type factory struct{}

func (factory) ID() string   { return "ndeval_opengl" }
func (factory) Name() string { return "OpenGL" }
func (factory) Weight() int  { return 20 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return fmt.Errorf("%w: apple uses metal", driver.ErrIncompatible)
	}
	if err := ffiopengl.ComputeAvailable(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (ndarray.Evaluator, error) {
	device, err := ffiopengl.OpenDevice()
	if err != nil {
		return nil, err
	}
	return &gpuEvaluator{device: device, own: true}, nil
}

type gpuEvaluator struct {
	device *ffiopengl.Device
	own    bool
	table  table.Table[*session]
}

func (g *gpuEvaluator) Name() string {
	if g == nil || g.device == nil {
		return "opengl"
	}
	return "opengl:" + g.device.Name()
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
		slog.Debug("ndeval opengl session", "device", g.device.Name())
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
	var gpu *ffiopengl.Device
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
		slog.Debug("ndeval opengl close", "device", gpu.Name(), "sessions", len(sessions))
		err = errors.Join(err, gpu.Close())
	}
	return err
}
