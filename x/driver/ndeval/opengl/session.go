package opengl

import (
	"context"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/run"
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
	if s == nil {
		return run.Eval[*ffiopengl.Buffer](ctx, true, nil, false, nil, false, nil, output, nil)
	}
	return run.Eval(ctx, true, s.kernel, s.device != nil, s.eval, s.prog != nil, &s.buf, output, s.dispatch)
}

func (s *session) dispatch() error {
	return run.Groups(s.prepare, &s.buf, s.kernel, func(bound []*ffiopengl.Buffer, push []byte, groups uint32) error {
		return s.device.Run(s.prog, groups, bound, push)
	})
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
		slog.Debug("ndeval opengl buffer grow", "buffers", grew, "bytes", total)
	}
	return nil
}

func (s *session) Close() error {
	if s == nil {
		return nil
	}
	return run.Release(s.eval, &s.forget, &s.prog, &s.buf)
}
