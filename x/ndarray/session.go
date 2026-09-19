package ndarray

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Session runs a compiled kernel repeatedly. Buffers grow only when the
// working set gets bigger; SPIR-V and the pipeline are created once.
type Session struct {
	kernel *Kernel
	device *vulkan.Device
	output *vulkan.Buffer
	inputs []*vulkan.Buffer
}

// Attach binds kernel to device. device may be nil for CPU-only EvalInto.
func (k *Kernel) Attach(ctx context.Context, device *vulkan.Device) (*Session, error) {
	if k == nil {
		return nil, ErrOp
	}
	s := &Session{kernel: k, device: device, inputs: make([]*vulkan.Buffer, len(k.bufs))}
	if device == nil {
		return s, nil
	}
	if _, err := k.ensurePipeline(ctx, device); err != nil {
		return nil, err
	}
	return s, nil
}

// Run writes inputs, dispatches, and reads output. output must already have length k.size.
func (s *Session) Run(ctx context.Context, output []float32) error {
	if s == nil || s.kernel == nil {
		return ErrOp
	}
	if s.device == nil {
		return s.kernel.EvalInto(output)
	}
	if len(output) < s.kernel.size {
		return fmt.Errorf("%w: output %d < %d", ErrSize, len(output), s.kernel.size)
	}
	if err := s.fit(); err != nil {
		return err
	}
	for i, b := range s.kernel.bufs {
		source := []float32(nil)
		if b != nil {
			source = b.data
		}
		if err := s.inputs[i].Write(floatView(source)); err != nil {
			return err
		}
	}
	if err := s.kernel.Run(ctx, s.device, s.output, s.inputs...); err != nil {
		return err
	}
	return s.output.Read(floatView(output[:s.kernel.size]))
}

func (s *Session) fit() error {
	if err := s.grow(&s.output, max(s.kernel.size, 1)*4); err != nil {
		return err
	}
	for i, b := range s.kernel.bufs {
		n := 1
		if b != nil {
			n = max(len(b.data), 1)
		}
		if err := s.grow(&s.inputs[i], n*4); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) grow(slot **vulkan.Buffer, bytes int) error {
	cur := *slot
	if cur != nil && cur.Len() >= bytes {
		return nil
	}
	b, err := s.device.Buffer(bytes)
	if err != nil {
		return err
	}
	slog.Debug("ndarray buffer grow", "bytes", bytes)
	if cur != nil {
		if err := cur.Close(); err != nil {
			return errors.Join(err, b.Close())
		}
	}
	*slot = b
	return nil
}

func floatView(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(v))), len(v)*4)
}

// Close frees session buffers. The kernel is left open.
func (s *Session) Close() error {
	if s == nil {
		return nil
	}
	var err error
	if s.output != nil {
		err = s.output.Close()
		s.output = nil
	}
	for i, b := range s.inputs {
		if b == nil {
			continue
		}
		if e := b.Close(); err == nil {
			err = e
		}
		s.inputs[i] = nil
	}
	return err
}
