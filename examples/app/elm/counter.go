package main

import (
	"image"
	"strconv"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
)

// Counter is the Elm example: one int, two buttons.
type Counter struct {
	gui.Dirty
	size  image.Point
	count int
	mode  daynight.Mode
	minus *gui.Box
	plus  *gui.Box
}

// NewCounter returns a counter at 0. The error is always nil.
func NewCounter() (*Counter, error) {
	return &Counter{size: image.Pt(400, 200), mode: daynight.Dark}, nil
}

func (counter *Counter) Init() gui.Cmd { return gui.Tick() }

func (counter *Counter) Update(msg gui.Msg) (gui.Model, gui.Cmd) {
	if counter == nil {
		return counter, nil
	}
	if mode, ok := msg.(gui.ModeMsg); ok {
		counter.Dirty, counter.mode = gui.See(counter.Dirty, counter.mode, mode.Mode)
	}
	if pointer, ok := msg.(window.Pointer); ok && pointer.Button == 1 && pointer.Pressed {
		if counter.minus != nil && counter.minus.Contains(pointer.Pos) {
			counter.Dirty, counter.count = gui.See(counter.Dirty, counter.count, counter.count-1)
		}
		if counter.plus != nil && counter.plus.Contains(pointer.Pos) {
			counter.Dirty, counter.count = gui.See(counter.Dirty, counter.count, counter.count+1)
		}
	}
	if size, ok := sizeOf(msg); ok && size.X > 0 && size.Y > 0 {
		counter.Dirty, counter.size = gui.See(counter.Dirty, counter.size, size)
	}
	return counter, nil
}

func (counter *Counter) View() gui.Node {
	if counter == nil {
		return nil
	}
	background, ink := gui.Palette(counter.mode)
	counter.minus = counter.button("-", gui.RGB{180, 70, 80, 255}, ink)
	counter.plus = counter.button("+", gui.RGB{70, 160, 100, 255}, ink)
	// Root Box fills the window; Align centers the packed Row.
	return &gui.Box{
		Fill:  &background,
		Align: gui.Alignment{0.5, 0.5},
		Child: gui.Row(
			counter.minus,
			&gui.Box{Width: 24},
			&gui.Box{
				Width:  100,
				Height: 56,
				Align:  gui.Alignment{0.5, 0.5},
				Child:  &gui.Text{Value: strconv.Itoa(counter.count), Ink: ink},
			},
			&gui.Box{Width: 24},
			counter.plus,
		),
	}
}

func (*Counter) button(label string, fill, ink gui.RGB) *gui.Box {
	return &gui.Box{
		Width:  56,
		Height: 56,
		Radius: 12,
		Fill:   &fill,
		Align:  gui.Alignment{0.5, 0.5},
		Child:  &gui.Text{Value: label, Ink: ink},
	}
}

func sizeOf(msg gui.Msg) (image.Point, bool) {
	switch message := msg.(type) {
	case gui.TickMsg:
		return message.Size, true
	case window.Resize:
		return message.Size, true
	default:
		return image.Point{}, false
	}
}
