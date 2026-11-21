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

func (counter *Counter) Init() Cmd { return Tick() }

func (counter *Counter) Update(msg Msg) (Model, Cmd) {
	if counter == nil {
		return counter, nil
	}
	if pointer, ok := msg.(window.Pointer); ok && pointer.Button == 1 && pointer.Pressed {
		if counter.minus != nil && counter.minus.Contains(pointer.Pos) {
			counter.count--
		}
		if counter.plus != nil && counter.plus.Contains(pointer.Pos) {
			counter.count++
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		counter.size = size
	}
	return counter, nil
}

func (counter *Counter) View() Node {
	if counter == nil {
		return nil
	}
	counter.minus = counter.button("-", Color{180, 70, 80, 255})
	counter.plus = counter.button("+", Color{70, 160, 100, 255})
	// Root Box fills the window; Align centers the packed Row.
	return &Box{
		Fill:  &Color{28, 28, 34, 255},
		Align: Alignment{0.5, 0.5},
		Child: Row(
			counter.minus,
			&Box{Width: 24},
			&Box{
				Width:  100,
				Height: 56,
				Align:  Alignment{0.5, 0.5},
				Child:  &Text{Value: strconv.Itoa(counter.count)},
			},
			&Box{Width: 24},
			counter.plus,
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
