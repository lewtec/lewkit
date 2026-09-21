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

func (stack *Stack) Layout(constraints BoxConstraints) Size {
	if stack == nil {
		return Size{}
	}
	stack.size = constraints.Constrain(Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	inner := Tight(stack.size.Width, stack.size.Height).Loosen()
	for _, node := range stack.Children {
		if node != nil {
			node.Layout(inner)
		}
	}
	return stack.size
}

func (stack *Stack) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if stack == nil {
		return accumulatorOf(picture)
	}
	if stack.Clip {
		clip = clip.Intersect(Rect{origin.X, origin.Y, stack.size.Width, stack.size.Height})
	}
	accumulator := accumulatorOf(picture)
	for _, node := range stack.Children {
		if node != nil {
			accumulator = node.Paint(origin, clip, picture)
		}
	}
	return accumulator
}

func (positioned *Positioned) Layout(constraints BoxConstraints) Size {
	if positioned == nil || positioned.Child == nil {
		return Size{}
	}
	inner := constraints.Loosen()
	inner.MaxHeight = unbounded
	positioned.size = positioned.Child.Layout(inner)
	return positioned.size
}

func (positioned *Positioned) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if positioned == nil || positioned.Child == nil {
		return accumulatorOf(picture)
	}
	return positioned.Child.Paint(origin.Add(Offset{positioned.X, positioned.Y}), clip, picture)
}
