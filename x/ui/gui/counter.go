package gui

import (
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
)

// Counter is the Elm example: one int, two buttons.
type Counter struct {
	Dirty
	size  image.Point
	count int
	mode  daynight.Mode
	minus *Box
	plus  *Box
}

// NewCounter returns a counter at 0. The error is always nil.
func NewCounter() (*Counter, error) {
	return &Counter{size: image.Pt(400, 200), mode: daynight.Dark}, nil
}

func (counter *Counter) Init() Cmd { return Tick() }

func (counter *Counter) Update(msg Msg) (Model, Cmd) {
	if counter == nil {
		return counter, nil
	}
	if mode, ok := msg.(ModeMsg); ok {
		Set(&counter.Dirty, &counter.mode, mode.Mode)
	}
	if pointer, ok := msg.(window.Pointer); ok && pointer.Button == 1 && pointer.Pressed {
		if counter.minus != nil && counter.minus.Contains(pointer.Pos) {
			Set(&counter.Dirty, &counter.count, counter.count-1)
		}
		if counter.plus != nil && counter.plus.Contains(pointer.Pos) {
			Set(&counter.Dirty, &counter.count, counter.count+1)
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		Set(&counter.Dirty, &counter.size, size)
	}
	return counter, nil
}

func (counter *Counter) View() Node {
	if counter == nil {
		return nil
	}
	background, ink := Palette(counter.mode)
	counter.minus = counter.button("-", Color{180, 70, 80, 255}, ink)
	counter.plus = counter.button("+", Color{70, 160, 100, 255}, ink)
	// Root Box fills the window; Align centers the packed Row.
	return &Box{
		Fill:  &background,
		Align: Alignment{0.5, 0.5},
		Child: Row(
			counter.minus,
			&Box{Width: 24},
			&Box{
				Width:  100,
				Height: 56,
				Align:  Alignment{0.5, 0.5},
				Child:  &Text{Value: strconv.Itoa(counter.count), Ink: ink},
			},
			&Box{Width: 24},
			counter.plus,
		),
	}
}

func (*Counter) button(label string, fill, ink Color) *Box {
	return &Box{
		Width:  56,
		Height: 56,
		Radius: 12,
		Fill:   &fill,
		Align:  Alignment{0.5, 0.5},
		Child:  &Text{Value: label, Ink: ink},
	}
}
