package gui

import (
	"github.com/lewtec/lewkit/x/ndarray"
	"golang.org/x/image/font"
)

// Node is one layout box. Layout is CPU: constraints down, [Size] up.
// Paint is the same shape: clip down, over-composite tensor up.
type Node interface {
	Layout(BoxConstraints) Size
	Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32]
}

type textRun struct {
	box    Rect
	body   []rune
	face   font.Face
	cursor int
	caret  bool
	ink    Color
}

// Color is straight RGBA 0..255.
type Color struct {
	Red, Green, Blue, Alpha uint8
}

// EdgeInsets is padding.
type EdgeInsets struct {
	Left, Top, Right, Bottom float32
}

func (insets EdgeInsets) Horizontal() float32 { return insets.Left + insets.Right }
func (insets EdgeInsets) Vertical() float32   { return insets.Top + insets.Bottom }

// Alignment is child placement in leftover space. 0 is start, 0.5 is center.
type Alignment struct {
	X, Y float32
}

// Draw is one rounded-rect fill in parent coordinates.
type Draw struct {
	X, Y, Width, Height     float32
	Red, Green, Blue, Alpha float32
	Radius                  float32
	ClipX, ClipY            float32
	ClipWidth, ClipHeight   float32
}
