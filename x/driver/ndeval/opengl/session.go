package opengl

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	ffiopengl "github.com/lewtec/lewkit/x/ffi/native/opengl"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to an OpenGL compute device.
type session struct {
	kernel *ndarray.Kernel
	device *ffiopengl.Device
	prog   *ffiopengl.Program
	buf    slots.Set[*ffiopengl.Buffer]
	eval   *sync.Mutex
	forget func()
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device *ffiopengl.Device) (*session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	src, err := glslSource(kernel, device.GLES())
	if err != nil {
		return nil, err
	}
	prog, err := device.Compile(src)
	if err != nil {
		return nil, err
	}
	slog.Debug("ndeval opengl compile", "bytes", len(src), "gles", device.GLES())
	return &session{
		kernel: kernel,
		device: device,
		prog:   prog,
		buf: slots.Set[*ffiopengl.Buffer]{
			Inputs: make([]*ffiopengl.Buffer, kernel.InputCount()),
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
	if s.prog == nil {
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
	groups := s.kernel.Groups()
	if groups < 1 {
		return nil
	}
	return s.device.Run(s.prog, groups, s.buf.Bound, push)
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
		slog.Debug("ndeval opengl buffer grow", "buffers", grew, "bytes", total)
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
	if s.prog != nil {
		err = s.prog.Close()
		s.prog = nil
	}
	return errors.Join(err, s.buf.Close())
}
