package gui

import "github.com/lewtec/lewkit/x/ndarray"

// Stage is the window root. Scene paints the whole frame, including
// navbars and notches. Child is laid out in the alive box from [Detect]
// and clipped to it, so text stays where the host can show it.
// [Picture.Render] wraps a root that is not a Stage. A [Raster] is the
// scene. A [Stack] lifts its direct [Raster] children into the scene.
// Any other root is the child.
type Stage struct {
	Scene Node
	Child Node

	size  Size
	alive Rect
}

// Shown is the alive box from the last Layout, in local coordinates.
func (stage *Stage) Shown() Rect {
	if stage == nil {
		return Rect{}
	}
	return stage.alive
}

func (stage *Stage) Layout(constraints BoxConstraints) Size {
	if stage == nil {
		return Size{}
	}
	size := constraints.Constrain(Size{Width: constraints.MaxWidth, Height: constraints.MaxHeight})
	stage.size = size
	full := Tight(size.Width, size.Height)
	full.scale = constraints.scale
	if stage.Scene != nil {
		stage.Scene.Layout(full)
	}
	stage.alive = Detect(Rect{Width: size.Width, Height: size.Height}, constraints.dead)
	if stage.Child != nil && stage.alive.Width > 0 && stage.alive.Height > 0 {
		tight := Tight(stage.alive.Width, stage.alive.Height)
		tight.scale = constraints.scale
		stage.Child.Layout(tight)
	}
	return size
}

func (stage *Stage) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if stage == nil {
		return accumulatorOf(picture)
	}
	accumulator := accumulatorOf(picture)
	if stage.Scene != nil {
		frame := Rect{origin.X, origin.Y, stage.size.Width, stage.size.Height}
		accumulator = stage.Scene.Paint(origin, clip.Intersect(frame), picture)
	}
	if stage.Child == nil || stage.alive.Width <= 0 || stage.alive.Height <= 0 {
		return accumulator
	}
	shown := Rect{origin.X + stage.alive.X, origin.Y + stage.alive.Y, stage.alive.Width, stage.alive.Height}
	next := clip.Intersect(shown)
	if next.Width <= 0 || next.Height <= 0 {
		return accumulator
	}
	return stage.Child.Paint(origin.Add(Offset{stage.alive.X, stage.alive.Y}), next, picture)
}

// stageRoot makes every painted view honor the alive box.
func stageRoot(root Node) Node {
	switch node := root.(type) {
	case *Stage:
		return node
	case *Raster:
		return &Stage{Scene: node}
	case *Stack:
		return stageStack(node)
	default:
		return &Stage{Child: root}
	}
}

func stageStack(stack *Stack) Node {
	if stack == nil {
		return &Stage{}
	}
	var scene []Node
	rest := make([]Node, 0, len(stack.Children))
	for _, child := range stack.Children {
		if _, ok := child.(*Raster); ok {
			scene = append(scene, child)
			continue
		}
		rest = append(rest, child)
	}
	if len(scene) == 0 {
		return &Stage{Child: stack}
	}
	backdrop := scene[0]
	if len(scene) > 1 {
		backdrop = &Stack{Children: scene}
	}
	var child Node
	if len(rest) > 0 {
		// A stack child is loose. Tight alive constraints would stretch a
		// small overlay to the whole frame.
		child = &Stack{Children: rest, Clip: stack.Clip}
	}
	return &Stage{Scene: backdrop, Child: child}
}

// Device lays Child out in device pixels. The window scale does not
// multiply its lengths. Use it when those lengths already follow the
// framebuffer, as a poster or a responsive layout does.
type Device struct {
	Child Node
}

func (node *Device) Layout(constraints BoxConstraints) Size {
	if node == nil || node.Child == nil {
		return Size{}
	}
	constraints.scale = 1
	return node.Child.Layout(constraints)
}

func (node *Device) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if node == nil || node.Child == nil {
		return accumulatorOf(picture)
	}
	return node.Child.Paint(origin, clip, picture)
}
