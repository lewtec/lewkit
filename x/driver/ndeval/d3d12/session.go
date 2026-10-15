package d3d12

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/run"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	ffid3d12 "github.com/lewtec/lewkit/x/ffi/native/d3d12"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to a Direct3D 12 device.
type session struct {
	kernel *ndarray.Kernel
	device *ffid3d12.Device
	pipe   *ffid3d12.Pipeline
	buf    slots.Set[*ffid3d12.Buffer]
	eval   *sync.Mutex
	forget func()
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device *ffid3d12.Device) (*session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	hlsl, threads, err := hlslSource(kernel)
	if err != nil {
		return nil, err
	}
	pipe, err := device.Compile(hlsl, threads)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndeval d3d12 compile", "bytes", len(hlsl), "threads", threads)
	return &session{
		kernel: kernel,
		device: device,
		pipe:   pipe,
		buf: slots.Set[*ffid3d12.Buffer]{
			Inputs: make([]*ffid3d12.Buffer, kernel.InputCount()),
		},
	}, nil
}

func (s *session) Eval(ctx context.Context, output []byte) error {
	if s == nil {
		return run.Eval[*ffid3d12.Buffer](ctx, true, nil, false, nil, false, nil, output, nil)
	}
	return run.Eval(ctx, true, s.kernel, s.device != nil, s.eval, s.pipe != nil, &s.buf, output, s.dispatch)
}

func (s *session) dispatch() error {
	return run.Groups(s.prepare, &s.buf, s.kernel, s.pipe.Dispatch)
}

func (s *session) prepare() error {
	return run.Prepare(s.fit, &s.buf, s.kernel)
}

func (s *session) fit() error {
	grew, total, err := s.buf.Fit(s.kernel, s.device.Buffer)
	if err != nil {
		return err
	}
	if grew > 0 {
		slog.Debug("ndeval d3d12 buffer grow", "buffers", grew, "bytes", total)
	}
	return nil
}

func (s *session) Close() error {
	if s == nil {
		return nil
	}
	return run.Release(s.eval, &s.forget, &s.pipe, &s.buf)
}
