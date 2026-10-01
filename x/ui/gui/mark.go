package gui

import (
	"image"

	"github.com/lewtec/lewkit/x/ndarray"
	"golang.org/x/image/font"
)

// MarkKind selects the fields of a [Mark] that are set.
// The zero value is not a mark.
type MarkKind uint8

const (
	MarkFill MarkKind = 1 + iota
	MarkText
	MarkImage
	MarkBackdrop
)

// Mark is one draw. [Picture] stores marks in paint order.
//
// A fill uses Box, Clip, Radius, and Color.
// A text run uses Box, Clip, Color, Text, Face, Cursor, and Caret.
// The caret is part of the run because it is drawn between glyphs.
// An image uses Box, Clip, Radius, and Src.
// A backdrop uses Pixels and covers the frame from this point under later marks.
type Mark struct {
	Kind   MarkKind
	Box    Rect
	Clip   Rect
	Radius float32
	Color  RGB
	Text   string
	Face   font.Face
	Src    image.Image
	Pixels *ndarray.Tensor[float32]
	Cursor int
	Caret  bool
}

// Fill records a rounded rectangle in paint order.
// The returned tensor is the software composite after this fill.
func (picture *Picture) Fill(box Rect, radius float32, color RGB, clip Rect) *ndarray.Tensor[float32] {
	if picture == nil {
		return nil
	}
	picture.marks = append(picture.marks, Mark{
		Kind:   MarkFill,
		Box:    box,
		Clip:   clip,
		Radius: radius,
		Color:  color,
	})
	return picture.over(Draw{
		X: box.X, Y: box.Y, Width: box.Width, Height: box.Height,
		Red: float32(color.Red), Green: float32(color.Green), Blue: float32(color.Blue), Alpha: float32(color.Alpha),
		Radius:     radius,
		ClipX:      clip.X,
		ClipY:      clip.Y,
		ClipWidth:  clip.Width,
		ClipHeight: clip.Height,
	})
}

// Text records one glyph run in paint order.
// A zero [RGB] is white, which is the ink [Text] draws when Ink is unset.
func (picture *Picture) Text(body string, face font.Face, color RGB, box, clip Rect, cursor int, caret bool) {
	if picture == nil {
		return
	}
	if color == (RGB{}) {
		color = RGB{255, 255, 255, 255}
	}
	picture.marks = append(picture.marks, Mark{
		Kind:   MarkText,
		Box:    box,
		Clip:   clip,
		Color:  color,
		Text:   body,
		Face:   face,
		Cursor: cursor,
		Caret:  caret,
	})
	picture.glyph(textRun{
		box:    box,
		clip:   clip,
		body:   []rune(body),
		face:   face,
		cursor: cursor,
		caret:  caret,
		ink:    color,
	})
}

// Image records src scaled into box. Radius punches a rounded mask.
func (picture *Picture) Image(src image.Image, box, clip Rect, radius float32) {
	if picture == nil || src == nil {
		return
	}
	picture.marks = append(picture.marks, Mark{
		Kind:   MarkImage,
		Box:    box,
		Clip:   clip,
		Radius: radius,
		Src:    src,
	})
	picture.blit(imageStamp{src: src, box: box, clip: clip, radius: radius})
}

// Backdrop records a frame-sized tensor under the marks that follow it.
func (picture *Picture) Backdrop(pixels *ndarray.Tensor[float32]) {
	if picture == nil || pixels == nil {
		return
	}
	picture.marks = append(picture.marks, Mark{Kind: MarkBackdrop, Pixels: pixels})
	picture.useRaster(pixels)
}

// Marks returns a copy of the draws recorded since the last [Picture.Render].
func (picture *Picture) Marks() []Mark {
	if picture == nil || len(picture.marks) == 0 {
		return nil
	}
	out := make([]Mark, len(picture.marks))
	copy(out, picture.marks)
	return out
}
