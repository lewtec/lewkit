package ndarray

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Vulkan is the evaluator for a Vulkan device. Sessions are reused per kernel.
type Vulkan struct {
	Device   *vulkan.Device
	own      bool
	mu       sync.Mutex
	sessions map[*Kernel]*Session
}

var _ Evaluator = (*Vulkan)(nil)
var _ driver.DriverFactory[Evaluator] = vulkanFactory{}

func init() {
	driver.Register[Evaluator](vulkanFactory{})
}

type vulkanFactory struct{}

func (vulkanFactory) ID() string   { return "ndarray_vulkan" }
func (vulkanFactory) Name() string { return "Vulkan" }
func (vulkanFactory) Weight() int  { return 50 }

var (
	vulkanProbeMu  sync.Mutex
	vulkanProbe    *vulkan.Device
	vulkanProbeErr error
	vulkanProbed   bool
)

func (vulkanFactory) CheckCompatibility(ctx context.Context) error {
	vulkanProbeMu.Lock()
	defer vulkanProbeMu.Unlock()
	if vulkanProbed {
		slog.Debug("vulkan probe reuse", "err", vulkanProbeErr)
		return vulkanProbeErr
	}
	vulkanProbed = true
	slog.Debug("vulkan probe open")
	vulkanProbe, vulkanProbeErr = vulkan.Open(ctx)
	if vulkanProbeErr != nil {
		slog.Debug("vulkan probe failed", "err", vulkanProbeErr)
	} else {
		slog.Debug("vulkan probe ok", "device", vulkanProbe.Name())
	}
	return vulkanProbeErr
}

func (vulkanFactory) New(ctx context.Context) (Evaluator, error) {
	vulkanProbeMu.Lock()
	d, err := vulkanProbe, vulkanProbeErr
	vulkanProbe = nil
	vulkanProbeMu.Unlock()
	if err != nil {
		return nil, err
	}
	if d == nil {
		slog.Debug("vulkan open")
		d, err = vulkan.Open(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		slog.Debug("vulkan take probe", "device", d.Name())
	}
	return &Vulkan{Device: d, own: true}, nil
}

func (v *Vulkan) Run(ctx context.Context, k *Kernel, output []float32, inputs [][]float32) error {
	if v == nil || v.Device == nil {
		return ErrOp
	}
	if k == nil {
		return ErrOp
	}
	v.mu.Lock()
	if v.sessions == nil {
		v.sessions = make(map[*Kernel]*Session)
	}
	s := v.sessions[k]
	v.mu.Unlock()
	if s == nil {
		var err error
		s, err = k.Attach(ctx, v.Device)
		if err != nil {
			return err
		}
		v.mu.Lock()
		if existing := v.sessions[k]; existing != nil {
			v.mu.Unlock()
			if err := s.Close(); err != nil {
				return err
			}
			s = existing
		} else {
			v.sessions[k] = s
			v.mu.Unlock()
			slog.Debug("vulkan session", "device", v.Device.Name())
		}
	}
	return s.Run(ctx, output, inputs)
}

func (v *Vulkan) Close() error {
	if v == nil {
		return nil
	}
	v.mu.Lock()
	sessions := v.sessions
	v.sessions = nil
	own, d := v.own, v.Device
	v.own = false
	v.Device = nil
	v.mu.Unlock()
	var err error
	for _, s := range sessions {
		err = errors.Join(err, s.Close())
	}
	if own && d != nil {
		slog.Debug("vulkan close", "device", d.Name(), "sessions", len(sessions))
		err = errors.Join(err, d.Close())
	}
	return err
}
