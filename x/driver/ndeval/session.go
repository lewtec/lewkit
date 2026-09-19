package ndeval

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	ffivulkan "github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/wasm/glsl"
)

// session is one kernel bound to a device: SPIR-V pipeline and GPU buffers.
type session struct {
	kernel  *ndarray.Kernel
	device  *ffivulkan.Device
	shader  *ffivulkan.Shader
	output  *ffivulkan.Buffer
	inputs  []*ffivulkan.Buffer
	bound   []*ffivulkan.Buffer
	staging []byte
	push    []byte
}

func newSession(ctx context.Context, kernel *ndarray.Kernel, device *ffivulkan.Device) (*session, error) {
	if kernel == nil || device == nil {
		return nil, ndarray.ErrOp
	}
	src, err := kernel.GLSL()
	if err != nil {
		return nil, err
	}
	spirv, err := glsl.Load(ctx, []byte(src))
	if err != nil {
		return nil, err
	}
	slog.Debug("ndeval spirv", "bytes", len(spirv))
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

func (s *session) Eval(ctx context.Context, output []byte) error {
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	size := s.kernel.Size()
	hostBytes := s.kernel.OutputBytes()
	if len(output) < hostBytes {
		return fmt.Errorf("%w: output %d < %d", ndarray.ErrSize, len(output), hostBytes)
	}
	if size == 0 {
		return nil
	}
	if err := s.fit(); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		if err := s.inputs[i].Write(s.inputBytes(i)); err != nil {
			return err
		}
	}
	need := s.kernel.Bindings()
	if cap(s.bound) < need {
		s.bound = make([]*ffivulkan.Buffer, need)
	} else {
		s.bound = s.bound[:need]
	}
	s.bound[0] = s.output
	copy(s.bound[1:], s.inputs)
	cmd, err := s.device.Begin()
	if err != nil {
		return err
	}
	if err := cmd.Bind(s.shader, s.bound...); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if cap(s.push) < ndarray.PushBytes {
		s.push = make([]byte, ndarray.PushBytes)
	} else {
		s.push = s.push[:ndarray.PushBytes]
	}
	s.kernel.FillPush(s.push)
	s.kernel.FillPush(s.push)
	if err := cmd.Push(s.push); err != nil {
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
	gpuBytes := size * 4
	if s.kernel.DType() == ndarray.F32 {
		return s.output.Read(output[:hostBytes])
	}
	if cap(s.staging) < gpuBytes {
		s.staging = make([]byte, gpuBytes)
	} else {
		s.staging = s.staging[:gpuBytes]
	}
	if err := s.output.Read(s.staging); err != nil {
		return err
	}
	s.copyHost(output[:hostBytes], size)
	return nil
}

func (s *session) copyHost(output []byte, size int) {
	if size == 0 || len(s.staging) < size*4 {
		return
	}
	floats := unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(s.staging))), size)
	switch s.kernel.DType() {
	case ndarray.U8:
		for i, v := range floats {
			output[i] = uint8(v)
		}
	case ndarray.I32:
		out := unsafe.Slice((*int32)(unsafe.Pointer(unsafe.SliceData(output))), size)
		for i, v := range floats {
			out[i] = int32(v)
		}
	}
}

func (s *session) fit() error {
	if err := s.grow(&s.output, max(s.kernel.Size(), 1)*4); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		n := max(s.gpuInputBytes(i), 1)
		if err := s.grow(&s.inputs[i], n); err != nil {
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

func (s *session) gpuInputBytes(i int) int {
	n := len(s.kernel.Input(i))
	if s.kernel.InputDType(i) == ndarray.U8 {
		return n * 4
	}
	return n
}

func (s *session) inputBytes(i int) []byte {
	raw := s.kernel.Input(i)
	if s.kernel.InputDType(i) != ndarray.U8 {
		return raw
	}
	need := len(raw) * 4
	if cap(s.staging) < need {
		s.staging = make([]byte, need)
	} else {
		s.staging = s.staging[:need]
		clear(s.staging)
	}
	for j, v := range raw {
		s.staging[j*4] = v
	}
	return s.staging
}
