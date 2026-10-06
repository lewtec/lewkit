package gui

import (
	"iter"
	"slices"

	"github.com/lewtec/lewkit/x/ndarray"
)

// inkStep is one glyph run or one image, in the order they were painted.
// image is -1 when the step is the text at text.
type inkStep struct {
	text  int
	image int
}

// lowered is the frame a screen plays, folded from a mark sequence.
// Fills keep their order. Text and images share order, so a line drawn
// after a picture stays on top of it. The backdrop is the last tensor.
type lowered struct {
	fills    []Draw
	texts    []textRun
	images   []imageStamp
	order    []inkStep
	backdrop *ndarray.Tensor[float32]
}

// lower folds marks into fills, text, images, and the last backdrop.
func lower(seq iter.Seq[Mark]) lowered {
	var out lowered
	if seq == nil {
		return out
	}
	for mark := range seq {
		switch mark.Kind {
		case MarkFill:
			out.fills = append(out.fills, Draw{
				X: mark.Box.X, Y: mark.Box.Y, Width: mark.Box.Width, Height: mark.Box.Height,
				Red: float32(mark.Color.Red), Green: float32(mark.Color.Green), Blue: float32(mark.Color.Blue), Alpha: float32(mark.Color.Alpha),
				Radius:    mark.Radius,
				ClipX:     mark.Clip.X,
				ClipY:     mark.Clip.Y,
				ClipWidth: mark.Clip.Width, ClipHeight: mark.Clip.Height,
			})
		case MarkText:
			out.order = append(out.order, inkStep{text: len(out.texts), image: -1})
			out.texts = append(out.texts, textRun{
				box:    mark.Box,
				clip:   mark.Clip,
				body:   []rune(mark.Text),
				face:   mark.Face,
				cursor: mark.Cursor,
				caret:  mark.Caret,
				ink:    mark.Color,
			})
		case MarkImage:
			if mark.Src == nil {
				break
			}
			out.order = append(out.order, inkStep{image: len(out.images)})
			out.images = append(out.images, imageStamp{src: mark.Src, box: mark.Box, clip: mark.Clip, radius: mark.Radius})
		case MarkBackdrop:
			if mark.Pixels != nil {
				out.backdrop = mark.Pixels
			}
		}
	}
	return out
}

func (picture *Picture) adopt(list lowered) {
	if picture == nil {
		return
	}
	picture.fills = list.fills
	picture.texts = list.texts
	picture.images = list.images
	picture.order = list.order
	picture.raster = list.backdrop
	picture.fillCount = len(list.fills)
}

// emit records one mark. A running iterator receives it.
// With no iterator, the picture lowers the recorded marks so readers see this prefix.
func (picture *Picture) emit(mark Mark) {
	if picture == nil || picture.stop {
		return
	}
	picture.marks = append(picture.marks, mark)
	if picture.yield == nil {
		picture.adopt(lower(slices.Values(picture.marks)))
		return
	}
	if !picture.yield(mark) {
		picture.stop = true
	}
}

// paintMarks streams the marks root paints at size, in order.
// Layout is the caller's. Ranging the sequence pulls the walk.
func (picture *Picture) paintMarks(root Node, size Size) iter.Seq[Mark] {
	return func(yield func(Mark) bool) {
		if picture == nil || root == nil {
			return
		}
		picture.yield = yield
		picture.stop = false
		defer func() {
			picture.yield = nil
			picture.stop = false
		}()
		root.Paint(Offset{}, Rect{0, 0, size.Width, size.Height}, picture)
	}
}
