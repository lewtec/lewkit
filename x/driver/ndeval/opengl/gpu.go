// Package opengl is the ndarray evaluator backed by an OpenGL compute device.
// It renders the kernel schedule as a GL 4.3 or OpenGL ES 3.1 compute shader.
// Apple stays on Metal, so this factory is incompatible on darwin and ios.
package opengl

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
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
	device   *ffiopengl.Device
	own      bool
	mu       sync.Mutex
	evalMu   sync.Mutex
	sessions map[*ndarray.Kernel]*session
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
	g.mu.Lock()
	if g.sessions == nil {
		g.sessions = make(map[*ndarray.Kernel]*session)
	}
	s := g.sessions[kernel]
	if s != nil && s.kernel == kernel {
		g.mu.Unlock()
		return s, nil
	}
	g.mu.Unlock()
	s, err := newSession(ctx, kernel, g.device)
	if err != nil {
		return nil, err
	}
	s.eval = &g.evalMu
	s.forget = func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		if g.sessions != nil && g.sessions[kernel] == s {
			delete(g.sessions, kernel)
		}
	}
	g.mu.Lock()
	if existing := g.sessions[kernel]; existing != nil && existing.kernel == kernel {
		g.mu.Unlock()
		if err := s.Close(); err != nil {
			return nil, err
		}
		return existing, nil
	}
	old := g.sessions[kernel]
	g.sessions[kernel] = s
	g.mu.Unlock()
	if old != nil {
		if err := old.Close(); err != nil {
			return nil, err
		}
	}
	slog.Debug("ndeval opengl session", "device", g.device.Name())
	return s, nil
}

func (g *gpuEvaluator) Close() error {
	if g == nil {
		return nil
	}
	g.mu.Lock()
	sessions := g.sessions
	g.sessions = nil
	own, gpu := g.own, g.device
	g.own = false
	g.device = nil
	g.mu.Unlock()
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
