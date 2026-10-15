package metal

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/run"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	ffimetal "github.com/lewtec/lewkit/x/ffi/native/metal"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to a Metal device.
type session struct {
	kernel *ndarray.Kernel
	device *ffimetal.Device
	pipe   *ffimetal.Pipeline
	buf    slots.Set[*ffimetal.Buffer]
	eval   *sync.Mutex
	forget func()
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device *ffimetal.Device) (*session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	msl, threads, err := mslSource(kernel)
	if err != nil {
		return nil, err
	}
	pipe, err := device.Compile(msl, threads)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndeval metal compile", "bytes", len(msl), "threads", threads)
	return &session{
		kernel: kernel,
		device: device,
		pipe:   pipe,
		buf: slots.Set[*ffimetal.Buffer]{
			Inputs: make([]*ffimetal.Buffer, kernel.InputCount()),
		},
	}, nil
}

func (s *session) Eval(ctx context.Context, output []byte) error {
	if s == nil {
		return run.Eval[*ffimetal.Buffer](ctx, true, nil, false, nil, false, nil, output, nil)
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
		slog.Debug("ndeval metal buffer grow", "buffers", grew, "bytes", total)
	}
	return nil
}

func (s *session) Close() error {
	if s == nil {
		return nil
	}
	return run.Release(s.eval, &s.forget, &s.pipe, &s.buf)
}
