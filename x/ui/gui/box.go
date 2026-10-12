package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/ndarray"
)

// Box is size, padding, alignment, optional fill, optional child.
// Width and Height are logical pixels. Zero takes the child, or the incoming
// max when there is no child. Once the text preference is above one, Height
// is only a minimum, so a larger face makes the box taller and the parent reflows.
type Box struct {
	Child     Node
	Padding   EdgeInsets
	Align     Alignment
	Width     float32
	Height    float32
	Fill      *RGB
	Radius    float32
	Clip      bool
	Key       string
	origin    Offset
	size      Size
	childSize Size
	scale     float32
}

func (box *Box) Layout(constraints BoxConstraints) Size {
	if box == nil {
		return Size{}
	}
	box.scale = constraints.scale
	pad := scaleInsets(box.Padding, box.scale)
	width, height := box.Width, box.Height
	if width > 0 {
		width *= factor(box.scale)
	}
	if height > 0 {
		height *= factor(box.scale)
	}
	inner := constraints.Deflate(pad)
	inner.MinWidth, inner.MinHeight = 0, 0
	if width > 0 {
		inner.MaxWidth = min(inner.MaxWidth, width)
	}
	// Preference one keeps an explicit height exact, including a child that
	// would overflow it. Above one, the face may need more lines, so the
	// height is a floor and the measured child can raise it.
	if height > 0 && factor(constraints.font) <= 1 {
		inner.MaxHeight = min(inner.MaxHeight, height)
	}
	var child Size
	if box.Child != nil {
		child = box.Child.Layout(inner)
	}
	want := Size{Width: width, Height: height}
	if want.Width == 0 {
		if box.Child != nil {
			want.Width = child.Width
		} else if height > 0 {
			want.Width = 0
		} else {
			want.Width = inner.MaxWidth
		}
	}
	if want.Height == 0 {
		if box.Child != nil {
			want.Height = child.Height
		} else if width > 0 {
			want.Height = 0
		} else {
			want.Height = inner.MaxHeight
		}
	} else if box.Child != nil && child.Height > want.Height {
		want.Height = child.Height
	}
	box.childSize = child
	box.size = constraints.Constrain(Size{
		Width:  want.Width + pad.Horizontal(),
		Height: want.Height + pad.Vertical(),
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
	if box.Key != "" && picture != nil {
		picture.noteKey(box.Key, bounds, clip)
	}
	accumulator := accumulatorOf(picture)
	pad := scaleInsets(box.Padding, box.scale)
	radius := box.Radius
	if radius > 0 {
		radius *= factor(box.scale)
	}
	if box.Fill != nil && picture != nil {
		accumulator = picture.Fill(bounds, radius, *box.Fill, clip)
	}
	if box.Child == nil {
		return accumulator
	}
	inner := Size{Width: box.size.Width - pad.Horizontal(), Height: box.size.Height - pad.Vertical()}
	childX := pad.Left + (inner.Width-box.childSize.Width)*box.Align.X
	childY := pad.Top + (inner.Height-box.childSize.Height)*box.Align.Y
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

// Unit is the pointer position inside the last painted box, 0 at the start and 1 at the end.
func (box *Box) Unit(position image.Point) (float32, float32) {
	if box == nil || box.size.Width < 1 || box.size.Height < 1 {
		return 0, 0
	}
	return (float32(position.X) - box.origin.X) / box.size.Width, (float32(position.Y) - box.origin.Y) / box.size.Height
}
