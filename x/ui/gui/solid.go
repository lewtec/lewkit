package gui

import "github.com/lewtec/lewkit/x/ndarray"

// Solid fills the frame with one RGBA color via [Picture].
type Solid struct {
	picture *Picture
	fill    Color
	size    Size
}

// NewSolid compiles the shared rounded-rect kernel with one slot.
func NewSolid(r, g, b, a uint8) (*Solid, error) {
	p, err := NewPicture(1)
	if err != nil {
		return nil, err
	}
	return &Solid{picture: p, fill: Color{r, g, b, a}, size: Size{1, 1}}, nil
}

func (s *Solid) Init() Cmd { return Tick() }

func (s *Solid) Update(msg Msg) (Model, Cmd) {
	if s == nil {
		return s, nil
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		s.size = Size{float32(size.X), float32(size.Y)}
	}
	return s, nil
}

func (s *Solid) frameSig() uint64 {
	if s == nil {
		return 0
	}
	return s.picture.frameSig()
}

func (s *Solid) View() *ndarray.Tensor[uint8] {
	if s == nil || s.picture == nil {
		return nil
	}
	t, err := s.picture.Render(&Box{Fill: &s.fill}, s.size)
	if err != nil {
		return s.picture.pixels
	}
	return t
}
