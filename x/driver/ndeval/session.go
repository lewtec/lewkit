package ndeval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to a device: SPIR-V pipeline and GPU buffers.
type session struct {
	kernel *ndarray.Kernel
	device *ffivulkan.Device
	shader *ffivulkan.Shader
	output *ffivulkan.Buffer
	inputs []*ffivulkan.Buffer
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device *ffivulkan.Device) (*session, error) {
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	spirv, err := kernel.SPIRV(ctx)
	if err != nil {
		return nil, err
	}
	push := kernel.Push()
	shader, err := device.Compile(ctx, ffivulkan.ShaderConfig{
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
		inputs: make([]*ffivulkan.Buffer, kernel.InputCount()),
	}, nil
}

func (s *session) Run(ctx context.Context, output []float32) error {
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	size := s.kernel.Size()
	if len(output) < size {
		return fmt.Errorf("%w: output %d < %d", ndarray.ErrSize, len(output), size)
	}
	if size == 0 {
		return nil
	}
	if err := s.fit(); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		if err := s.inputs[i].Write(floatView(s.kernel.Input(i))); err != nil {
			return err
		}
	}
	need := s.kernel.Bindings()
	bufs := make([]*ffivulkan.Buffer, need)
	bufs[0] = s.output
	copy(bufs[1:], s.inputs)
	cmd, err := s.device.Begin()
	if err != nil {
		return err
	}
	if err := cmd.Bind(s.shader, bufs...); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Push(s.kernel.Push()); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Dispatch(s.kernel.Groups(), 1, 1); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Submit(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
	}
	return s.output.Read(floatView(output[:size]))
}

func (s *session) fit() error {
	if err := s.grow(&s.output, max(s.kernel.Size(), 1)*4); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		n := max(len(s.kernel.Input(i)), 1)
		if err := s.grow(&s.inputs[i], n*4); err != nil {
			return err
		}
	}
	return nil
}

func (s *session) grow(slot **ffivulkan.Buffer, bytes int) error {
	cur := *slot
	if cur != nil && cur.Len() >= bytes {
		return nil
	}
	b, err := s.device.Buffer(bytes)
	if err != nil {
		return err
	}
	slog.Debug("ndeval buffer grow", "bytes", bytes)
	if cur != nil {
		if err := cur.Close(); err != nil {
			return errors.Join(err, b.Close())
		}
	}
	*slot = b
	return nil
}

func (s *session) Close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.shader != nil {
		err = s.shader.Close()
		s.shader = nil
	}
	if s.output != nil {
		err = errors.Join(err, s.output.Close())
		s.output = nil
	}
	for i, b := range s.inputs {
		if b == nil {
			continue
		}
		err = errors.Join(err, b.Close())
		s.inputs[i] = nil
	}
	return err
}

func floatView(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(v))), len(v)*4)
}
