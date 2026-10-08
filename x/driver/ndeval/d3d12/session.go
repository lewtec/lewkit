package d3d12

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	ffid3d12 "github.com/lewtec/lewkit/x/ffi/native/d3d12"
	"github.com/lewtec/lewkit/x/ndarray"
)

// session is one kernel bound to a Direct3D 12 device.
type session struct {
	kernel  *ndarray.Kernel
	device  *ffid3d12.Device
	pipe    *ffid3d12.Pipeline
	output  *ffid3d12.Buffer
	inputs  []*ffid3d12.Buffer
	bound   []*ffid3d12.Buffer
	staging []byte
	push    []byte
	eval    *sync.Mutex
	forget  func()
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
		inputs: make([]*ffid3d12.Buffer, kernel.InputCount()),
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
	size := s.kernel.Size()
	hostBytes := s.kernel.OutputBytes()
	if len(output) < hostBytes {
		return fmt.Errorf("%w: output %d < %d", ndarray.ErrSize, len(output), hostBytes)
	}
	if size == 0 {
		return nil
	}
	if err := s.dispatch(); err != nil {
		return err
	}
	if s.kernel.DType() != ndarray.U8 {
		return s.output.Read(output[:hostBytes])
	}
	gpuBytes := size * 4
	if cap(s.staging) < gpuBytes {
		s.staging = make([]byte, gpuBytes)
	} else {
		s.staging = s.staging[:gpuBytes]
	}
	if err := s.output.Read(s.staging); err != nil {
		return err
	}
	for i := 0; i < size && i < len(output); i++ {
		output[i] = uint8(binary.LittleEndian.Uint32(s.staging[i*4:]))
	}
	return nil
}

func (s *session) dispatch() error {
	if err := s.prepare(); err != nil {
		return err
	}
	if cap(s.push) < ndarray.PushBytes {
		s.push = make([]byte, ndarray.PushBytes)
	} else {
		s.push = s.push[:ndarray.PushBytes]
	}
	s.kernel.FillPush(s.push)
	groups := int(s.kernel.Groups())
	if groups < 1 {
		return nil
	}
	return s.pipe.Dispatch(s.bound, s.push, groups)
}

func (s *session) prepare() error {
	if err := s.fit(); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		if err := s.inputs[i].Store(s.inputBytes(i)); err != nil {
			return err
		}
	}
	need := s.kernel.Bindings()
	if cap(s.bound) < need {
		s.bound = make([]*ffid3d12.Buffer, need)
	} else {
		s.bound = s.bound[:need]
	}
	s.bound[0] = s.output
	copy(s.bound[1:], s.inputs)
	return nil
}

func (s *session) fit() error {
	var grew, total int
	if err := s.grow(&s.output, max(s.kernel.Size(), 1)*4, &grew, &total); err != nil {
		return err
	}
	for i := 0; i < s.kernel.InputCount(); i++ {
		n := max(s.gpuInputBytes(i), 1)
		if err := s.grow(&s.inputs[i], n, &grew, &total); err != nil {
			return err
		}
	}
	if grew > 0 {
		slog.Debug("ndeval d3d12 buffer grow", "buffers", grew, "bytes", total)
	}
	return nil
}

func (s *session) grow(slot **ffid3d12.Buffer, bytes int, grew, total *int) error {
	cur := *slot
	if cur != nil && cur.Len() >= bytes {
		return nil
	}
	b, err := s.device.Buffer(bytes)
	if err != nil {
		return err
	}
	*grew++
	*total += bytes
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
