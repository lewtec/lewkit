package gui

import (
	"github.com/lewtec/lewkit/x/ndarray"
	ndimage "github.com/lewtec/lewkit/x/ndarray/image"
)

// Solid fills the frame with one RGBA color.
type Solid struct {
	pixels *ndarray.Tensor[uint8]
}

// NewSolid builds a (1,1,4) color tensor. [Run] resizes it to the window.
func NewSolid(r, g, b, a uint8) (*Solid, error) {
	fill, err := ndimage.Fill(1, 1, float32(r), float32(g), float32(b), float32(a))
	if err != nil {
		return nil, err
	}
	return &Solid{pixels: fill.Cast[uint8]()}, nil
}

func (s *Solid) Init() Cmd { return nil }

func (s *Solid) Update(msg Msg) (Model, Cmd) {
	if s == nil || s.pixels == nil {
		return s, nil
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		_ = s.pixels.Resize(ndarray.Shape{size.Y, size.X, 4})
	}
	return s, nil
}

func (s *Solid) View() *ndarray.Tensor[uint8] {
	if s == nil {
		return nil
	}
	return s.pixels
}
