package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/driver/window"
)

// Drag follows one held button. A release inside the slop is a tap.
// A move past it reports how far the pointer traveled since the previous
// sample, in device pixels. Delta is previous minus current, so a finger
// moving up has a positive Y and the content can follow it.
type Drag struct {
	down   bool
	moved  bool
	origin image.Point
	last   image.Point
}

// DragStep is one sample from [Drag.Step]. Tap and Delta are alternatives.
type DragStep struct {
	Tap   bool
	At    image.Point
	Delta image.Point
}

// Slop is the movement, in device pixels, that still counts as a tap.
// scale is device pixels per logical pixel. A missing scale is one.
func Slop(scale float32) int {
	return max(int(8*density(scale)), 8)
}

// Logical converts a device-pixel distance to logical pixels.
// Scroll offsets that the layout multiplies by density use this.
func Logical(scale float32, device int) float32 {
	return float32(device) / density(scale)
}

func density(scale float32) float32 {
	return max(scale, 1)
}

// Step updates the gesture. The bool is false when the sample changes nothing.
func (d *Drag) Step(event window.Pointer, slop int) (DragStep, bool) {
	if d == nil {
		return DragStep{}, false
	}
	if slop < 0 {
		slop = 0
	}
	if event.Button == 1 && event.Pressed {
		d.down = true
		d.moved = false
		d.origin = event.Pos
		d.last = event.Pos
		return DragStep{}, false
	}
	if event.Button == 1 && !event.Pressed {
		tap := d.down && !d.moved
		at := event.Pos
		d.down = false
		d.moved = false
		if !tap {
			return DragStep{}, false
		}
		return DragStep{Tap: true, At: at}, true
	}
	if !d.down || event.Buttons&window.ButtonLeft == 0 {
		return DragStep{}, false
	}
	if !d.moved {
		dx := event.Pos.X - d.origin.X
		dy := event.Pos.Y - d.origin.Y
		if dx*dx+dy*dy <= slop*slop {
			d.last = event.Pos
			return DragStep{}, false
		}
		d.moved = true
	}
	delta := image.Pt(d.last.X-event.Pos.X, d.last.Y-event.Pos.Y)
	d.last = event.Pos
	if delta == (image.Point{}) {
		return DragStep{}, false
	}
	return DragStep{Delta: delta}, true
}
