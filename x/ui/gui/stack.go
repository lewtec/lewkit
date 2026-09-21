package gui

import "github.com/lewtec/lewkit/x/ndarray"

// Stack paints children on top of each other. Size is the incoming max.
type Stack struct {
	Children []Node
	Clip     bool

	size Size
}

// Positioned paints a child at an offset. Layout uses loose max constraints.
type Positioned struct {
	X, Y  float32
	Child Node

	size Size
}

func (s *Stack) Layout(c BoxConstraints) Size {
	if s == nil {
		return Size{}
	}
	s.size = c.Constrain(Size{Width: c.MaxWidth, Height: c.MaxHeight})
	inner := Tight(s.size.Width, s.size.Height).Loosen()
	for _, node := range s.Children {
		if node != nil {
			node.Layout(inner)
		}
	}
	return s.size
}

func (s *Stack) Paint(origin Offset, clip Rect, pic *Picture) *ndarray.Tensor[float32] {
	if s == nil {
		return accOf(pic)
	}
	if s.Clip {
		clip = clip.Intersect(Rect{origin.X, origin.Y, s.size.Width, s.size.Height})
	}
	acc := accOf(pic)
	for _, node := range s.Children {
		if node != nil {
			acc = node.Paint(origin, clip, pic)
		}
	}
	return acc
}

func (p *Positioned) Layout(c BoxConstraints) Size {
	if p == nil || p.Child == nil {
		return Size{}
	}
	inner := c.Loosen()
	inner.MaxHeight = unbounded
	p.size = p.Child.Layout(inner)
	return p.size
}

func (p *Positioned) Paint(origin Offset, clip Rect, pic *Picture) *ndarray.Tensor[float32] {
	if p == nil || p.Child == nil {
		return accOf(pic)
	}
	return p.Child.Paint(origin.Add(Offset{p.X, p.Y}), clip, pic)
}
