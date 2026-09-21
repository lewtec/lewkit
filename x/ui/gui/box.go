package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Box is size, padding, alignment, optional fill, optional child.
type Box struct {
	Child     Node
	Padding   EdgeInsets
	Align     Alignment
	Width     float32
	Height    float32
	Fill      *Color
	Radius    float32
	Clip      bool
	origin    Offset
	size      Size
	childSize Size
}

func (box *Box) Layout(constraints BoxConstraints) Size {
	if box == nil {
		return Size{}
	}
	inner := constraints.Deflate(box.Padding)
	inner.MinWidth, inner.MinHeight = 0, 0
	if box.Width > 0 {
		inner.MaxWidth = min(inner.MaxWidth, box.Width)
	}
	if box.Height > 0 {
		inner.MaxHeight = min(inner.MaxHeight, box.Height)
	}
	var child Size
	if box.Child != nil {
		child = box.Child.Layout(inner)
	}
	want := Size{Width: box.Width, Height: box.Height}
	if want.Width == 0 {
		if box.Child != nil {
			want.Width = child.Width
		} else if box.Height > 0 {
			want.Width = 0
		} else {
			want.Width = inner.MaxWidth
		}
	}
	if want.Height == 0 {
		if box.Child != nil {
			want.Height = child.Height
		} else if box.Width > 0 {
			want.Height = 0
		} else {
			want.Height = inner.MaxHeight
		}
	}
	box.childSize = child
	box.size = constraints.Constrain(Size{
		Width:  want.Width + box.Padding.Horizontal(),
		Height: want.Height + box.Padding.Vertical(),
	})
	return box.size
}

func (box *Box) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if box == nil {
		return accumulatorOf(picture)
	}
	box.origin = origin
	bounds := Rect{origin.X, origin.Y, box.size.Width, box.size.Height}
	if box.Clip {
		clip = clip.Intersect(bounds)
	}
	accumulator := accumulatorOf(picture)
	if box.Fill != nil && picture != nil {
		accumulator = picture.over(Draw{
			X: origin.X, Y: origin.Y, Width: box.size.Width, Height: box.size.Height,
			Red: float32(box.Fill.Red), Green: float32(box.Fill.Green), Blue: float32(box.Fill.Blue), Alpha: float32(box.Fill.Alpha),
			Radius:     box.Radius,
			ClipX:      clip.X,
			ClipY:      clip.Y,
			ClipWidth:  clip.Width,
			ClipHeight: clip.Height,
		})
	}
	if box.Child == nil {
		return accumulator
	}
	inner := Size{Width: box.size.Width - box.Padding.Horizontal(), Height: box.size.Height - box.Padding.Vertical()}
	childX := box.Padding.Left + (inner.Width-box.childSize.Width)*box.Align.X
	childY := box.Padding.Top + (inner.Height-box.childSize.Height)*box.Align.Y
	return box.Child.Paint(origin.Add(Offset{childX, childY}), clip, picture)
}

// Contains is true when position is inside the last painted box.
func (box *Box) Contains(position image.Point) bool {
	if box == nil {
		return false
	}
	x, y := float32(position.X), float32(position.Y)
	return x >= box.origin.X && x < box.origin.X+box.size.Width && y >= box.origin.Y && y < box.origin.Y+box.size.Height
}
