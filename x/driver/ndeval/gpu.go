package ndeval

import (
	"context"
	"errors"
	"image"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

type gpuFactory struct{}

func (gpuFactory) ID() string   { return "ndeval_vulkan" }
func (gpuFactory) Name() string { return "Vulkan" }
func (gpuFactory) Weight() int  { return 50 }

func (gpuFactory) CheckCompatibility(ctx context.Context) error {
	_, err := vulkan.List(ctx)
	return err
}

func (gpuFactory) New(ctx context.Context) (ndarray.Evaluator, error) {
	gpu, err := vulkan.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &gpuEvaluator{device: gpu, own: true}, nil
}

type gpuEvaluator struct {
	device   vulkan.Device
	own      bool
	mu       sync.Mutex
	sessions map[*ndarray.Kernel]*ndarray.Session
}

func newGPU(native *ffivulkan.Device, own bool) *gpuEvaluator {
	if native == nil {
		return &gpuEvaluator{}
	}
	return &gpuEvaluator{device: vulkan.Wrap(native), own: own}
}

func (g *gpuEvaluator) Name() string {
	if g == nil || g.device == nil {
		return "vulkan"
	}
	return "vulkan:" + g.device.Name()
}

func (g *gpuEvaluator) native() *ffivulkan.Device {
	if g == nil || g.device == nil {
		return nil
	}
	return g.device.Native()
}

func (g *gpuEvaluator) Run(ctx context.Context, kernel *ndarray.Kernel, output []float32) error {
	session, err := g.session(ctx, kernel)
	if err != nil {
		return err
	}
	return session.Run(ctx, output)
}

func (g *gpuEvaluator) RunRGBA(ctx context.Context, kernel *ndarray.Kernel, destination *image.RGBA) error {
	session, err := g.session(ctx, kernel)
	if err != nil {
		return err
	}
	return session.RunRGBA(ctx, destination)
}

func (g *gpuEvaluator) session(ctx context.Context, kernel *ndarray.Kernel) (*ndarray.Session, error) {
	native := g.native()
	if native == nil || kernel == nil {
		return nil, ndarray.ErrOp
	}
	g.mu.Lock()
	if g.sessions == nil {
		g.sessions = make(map[*ndarray.Kernel]*ndarray.Session)
	}
	s := g.sessions[kernel]
	g.mu.Unlock()
	if s != nil {
		return s, nil
	}
	s, err := kernel.Attach(ctx, native)
	if err != nil {
		return nil, err
	}
	g.mu.Lock()
	if existing := g.sessions[kernel]; existing != nil {
		g.mu.Unlock()
		if err := s.Close(); err != nil {
			return nil, err
		}
		return existing, nil
	}
	g.sessions[kernel] = s
	g.mu.Unlock()
	slog.Debug("ndeval vulkan session", "device", native.Name())
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
		slog.Debug("ndeval vulkan close", "device", gpu.Name(), "sessions", len(sessions))
		err = errors.Join(err, gpu.Close())
	}
	return err
}
