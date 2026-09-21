package gui

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

func (s *Stack) Paint(origin Offset, clip Rect, paint *painter) {
	if s == nil {
		return
	}
	if s.Clip {
		clip = clip.Intersect(Rect{origin.X, origin.Y, s.size.Width, s.size.Height})
	}
	for _, node := range s.Children {
		paintChild(node, origin, clip, paint)
	}
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

func (p *Positioned) Paint(origin Offset, clip Rect, out *painter) {
	if p == nil {
		return
	}
	paintChild(p.Child, origin.Add(Offset{p.X, p.Y}), clip, out)
}
