package gui

import (
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Counter is the Elm example: one int, two buttons.
type Counter struct {
	picture *Picture
	size    image.Point
	count   int
	minus   *Box
	plus    *Box
}

// NewCounter compiles the Picture kernel used by View.
func NewCounter() (*Counter, error) {
	p, err := NewPicture(8)
	if err != nil {
		return nil, err
	}
	return &Counter{picture: p, size: image.Pt(400, 200)}, nil
}

func (c *Counter) Init() Cmd { return Tick() }

func (c *Counter) Update(msg Msg) (Model, Cmd) {
	if c == nil {
		return c, nil
	}
	if p, ok := msg.(window.Pointer); ok && p.Button == 1 && p.Pressed {
		if c.minus.Contains(p.Pos) {
			c.count--
		}
		if c.plus.Contains(p.Pos) {
			c.count++
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		c.size = size
	}
	return c, nil
}

func (c *Counter) frameSig() uint64 {
	if c == nil {
		return 0
	}
	return c.picture.frameSig()
}

func (c *Counter) View() *ndarray.Tensor[uint8] {
	if c == nil || c.picture == nil {
		return nil
	}
	t, err := c.picture.Render(c.tree(), Size{float32(c.size.X), float32(c.size.Y)})
	if err != nil {
		return c.picture.pixels
	}
	return t
}

func (c *Counter) tree() Node {
	c.minus = roundButton("-", Color{180, 70, 80, 255})
	c.plus = roundButton("+", Color{70, 160, 100, 255})
	// Root Box fills the window; Align centers the packed Row.
	return &Box{
		Fill:  &Color{28, 28, 34, 255},
		Align: Alignment{0.5, 0.5},
		Child: Row(
			c.minus,
			&Box{Width: 24},
			&Box{
				Width:  100,
				Height: 56,
				Align:  Alignment{0.5, 0.5},
				Child:  &Text{Value: strconv.Itoa(c.count)},
			},
			&Box{Width: 24},
			c.plus,
		),
	}
}

func roundButton(label string, fill Color) *Box {
	return &Box{
		Width:  56,
		Height: 56,
		Radius: 12,
		Fill:   &fill,
		Align:  Alignment{0.5, 0.5},
		Child:  &Text{Value: label},
	}
}
