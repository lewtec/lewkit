package gui

import (
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/window"
)

// Counter is the Elm example: one int, two buttons.
type Counter struct {
	size  image.Point
	count int
	minus *Box
	plus  *Box
}

// NewCounter returns a counter at 0. The error is always nil.
func NewCounter() (*Counter, error) {
	return &Counter{size: image.Pt(400, 200)}, nil
}

func (c *Counter) Init() Cmd { return Tick() }

func (c *Counter) Update(msg Msg) (Model, Cmd) {
	if c == nil {
		return c, nil
	}
	if p, ok := msg.(window.Pointer); ok && p.Button == 1 && p.Pressed {
		if c.minus != nil && c.minus.Contains(p.Pos) {
			c.count--
		}
		if c.plus != nil && c.plus.Contains(p.Pos) {
			c.count++
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		c.size = size
	}
	return c, nil
}

func (c *Counter) View() Node {
	if c == nil {
		return nil
	}
	c.minus = c.button("-", Color{180, 70, 80, 255})
	c.plus = c.button("+", Color{70, 160, 100, 255})
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

func (*Counter) button(label string, fill Color) *Box {
	return &Box{
		Width:  56,
		Height: 56,
		Radius: 12,
		Fill:   &fill,
		Align:  Alignment{0.5, 0.5},
		Child:  &Text{Value: label},
	}
}
