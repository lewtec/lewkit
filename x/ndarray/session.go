package ndarray

import (
	"context"
	"errors"
	"fmt"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
)

// Session runs a compiled kernel repeatedly. Buffers grow only when the
// working set gets bigger; SPIR-V and the pipeline are created once.
type Session struct {
	k   *Kernel
	d   *vulkan.Device
	dst *vulkan.Buffer
	src []*vulkan.Buffer
}

// Attach binds k to d. d may be nil for CPU-only EvalInto.
func (k *Kernel) Attach(ctx context.Context, d *vulkan.Device) (*Session, error) {
	if k == nil {
		return nil, ErrOp
	}
	s := &Session{k: k, d: d, src: make([]*vulkan.Buffer, len(k.slots))}
	if d == nil {
		return s, nil
	}
	if _, err := k.shader(ctx, d); err != nil {
		return nil, err
	}
	return s, nil
}

// Run writes srcs, dispatches, and reads dst. dst must already have length k.n.
func (s *Session) Run(ctx context.Context, dst []float32, srcs [][]float32) error {
	if s == nil || s.k == nil {
		return ErrOp
	}
	if s.d == nil {
		return s.k.EvalInto(dst, srcs)
	}
	if len(dst) < s.k.n {
		return fmt.Errorf("%w: dst %d < %d", ErrSize, len(dst), s.k.n)
	}
	if len(srcs) != len(s.k.slots) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(s.k.slots), len(srcs))
	}
	if err := s.fit(dst, srcs); err != nil {
		return err
	}
	for i, src := range srcs {
		if err := s.src[i].Write(f32view(src)); err != nil {
			return err
		}
	}
	if err := s.k.Run(ctx, s.d, s.dst, s.src...); err != nil {
		return err
	}
	return s.dst.Read(f32view(dst[:s.k.n]))
}

func (s *Session) fit(dst []float32, srcs [][]float32) error {
	if err := s.grow(&s.dst, max(len(dst), 1)*4); err != nil {
		return err
	}
	for i, src := range srcs {
		if err := s.grow(&s.src[i], max(len(src), 1)*4); err != nil {
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
	b, err := s.d.Buffer(bytes)
	if err != nil {
		return err
	}
	if cur != nil {
		if err := cur.Close(); err != nil {
			return errors.Join(err, b.Close())
		}
	}
	*slot = b
	return nil
}

func f32view(v []float32) []byte {
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
	if s.dst != nil {
		err = s.dst.Close()
		s.dst = nil
	}
	for i, b := range s.src {
		if b == nil {
			continue
		}
		if e := b.Close(); err == nil {
			err = e
		}
		s.src[i] = nil
	}
	return err
}
