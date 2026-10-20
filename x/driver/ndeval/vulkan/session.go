package vulkan

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/run"
	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to a device: SPIR-V pipeline and GPU buffers.
type session struct {
	kernel *ndarray.Kernel
	device drvvulkan.Device
	shader *drvvulkan.Shader
	buf    slots.Set[*drvvulkan.Buffer]
	eval   *sync.Mutex
	forget func()
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device drvvulkan.Device) (*session, error) {
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	src, err := glslSource(kernel)
	if err != nil {
		return nil, err
	}
	spirv, err := compileCompute(ctx, []byte(src))
	if err != nil {
		return nil, err
	}
	slog.Debug("ndeval spirv", "bytes", len(spirv))
	push := kernel.Push()
	shader, err := device.Compile(ctx, drvvulkan.ShaderConfig{
		SPIRV:     spirv,
		Bindings:  kernel.Bindings(),
		PushBytes: len(push),
	})
	if err != nil {
		return nil, err
	}
	return &session{
		kernel: kernel,
		device: device,
		shader: shader,
		buf: slots.Set[*drvvulkan.Buffer]{
			Inputs: make([]*drvvulkan.Buffer, kernel.InputCount()),
		},
	}, nil
}

func (s *session) Eval(ctx context.Context, output []byte) error {
	if s == nil {
		return run.Eval[*drvvulkan.Buffer](ctx, false, nil, false, nil, false, nil, output, nil)
	}
	return run.Eval(ctx, false, s.kernel, s.device != nil, s.eval, s.shader != nil, &s.buf, output, s.dispatch)
}

func (s *session) dispatch() error {
	if err := s.prepare(); err != nil {
		return err
	}
	cmd, err := s.device.Begin()
	if err != nil {
		return err
	}
	if err := s.record(cmd); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Submit(); err != nil {
		return err
	}
	return cmd.Wait()
}

func (s *session) prepare() error {
	return run.Prepare(s.fit, &s.buf, s.kernel)
}

func (s *session) record(cmd *drvvulkan.Cmd) error {
	if err := cmd.Bind(s.shader, s.buf.Bound...); err != nil {
		return err
	}
	push := s.buf.FillPush(s.kernel)
	if err := cmd.Push(push); err != nil {
		return err
	}
	return cmd.Dispatch(s.kernel.Groups(), 1, 1)
}

func (s *session) fit() error {
	grew, total, err := s.buf.Fit(s.kernel, s.device.Buffer)
	if err != nil {
		return err
	}
	if grew > 0 {
		slog.Debug("ndeval buffer grow", "buffers", grew, "bytes", total)
	}
	return nil
}

func (s *session) Close() error {
	if s == nil {
		return nil
	}
	return run.Release(s.eval, &s.forget, &s.shader, &s.buf)
}
