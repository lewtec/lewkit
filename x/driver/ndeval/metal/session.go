package metal

import (
	"context"
	"errors"
	"log/slog"
	"sync"

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
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	if s.pipe == nil {
		return ndarray.ErrOp
	}
	if err := slots.CheckOutput(s.kernel, output); err != nil {
		return err
	}
	if s.kernel.Size() == 0 {
		return nil
	}
	if err := s.dispatch(); err != nil {
		return err
	}
	return s.buf.ReadHost(s.kernel, output)
}

func (s *session) dispatch() error {
	if err := s.prepare(); err != nil {
		return err
	}
	push := s.buf.FillPush(s.kernel)
	groups := int(s.kernel.Groups())
	if groups < 1 {
		return nil
	}
	return s.pipe.Dispatch(s.buf.Bound, push, groups)
}

func (s *session) prepare() error {
	if err := s.fit(); err != nil {
		return err
	}
	return s.buf.Load(s.kernel)
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
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	if s.forget != nil {
		s.forget()
		s.forget = nil
	}
	var err error
	if s.pipe != nil {
		err = s.pipe.Close()
		s.pipe = nil
	}
	return errors.Join(err, s.buf.Close())
}
