package vulkan

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
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
	spirv, err := glsl.Load(ctx, []byte(src))
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
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	if s.shader == nil {
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
	if err := s.fit(); err != nil {
		return err
	}
	return s.buf.Load(s.kernel)
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
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	if s.forget != nil {
		s.forget()
		s.forget = nil
	}
	var err error
	if s.shader != nil {
		err = s.shader.Close()
		s.shader = nil
	}
	return errors.Join(err, s.buf.Close())
}
