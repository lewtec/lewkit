package gui

import (
	"image"
	"image/color"
	"image/draw"

	lewimage "github.com/lewtec/lewkit/x/image"
	"github.com/lewtec/lewkit/x/ndarray"
	"golang.org/x/image/font"
)

var caretFill = &image.Uniform{C: color.RGBA{R: 220, G: 220, B: 220, A: 255}}

// Text is a layout node. Nil Face uses [lewimage.Face]. Cursor -1 hides the caret.
type Text struct {
	Value  string
	Face   font.Face
	Cursor int
	Caret  bool

	origin Offset
	size   Size
}

func (t *Text) face() font.Face {
	if t == nil {
		return lewimage.Face()
	}
	return lewimage.Use(t.Face)
}

func (t *Text) Layout(c BoxConstraints) Size {
	if t == nil {
		return Size{}
	}
	w, h := t.measure(c.MaxWidth)
	t.size = c.Constrain(Size{Width: w, Height: h})
	return t.size
}

func (t *Text) Paint(origin Offset, clip Rect, pic *Picture) *ndarray.Tensor[float32] {
	if t == nil {
		return accOf(pic)
	}
	t.origin = origin
	if pic != nil {
		box := Rect{origin.X, origin.Y, t.size.Width, t.size.Height}.Intersect(clip)
		pic.glyph(textRun{
			box:    box,
			body:   []rune(t.Value),
			face:   t.face(),
			cursor: t.Cursor,
			caret:  t.Caret,
		})
	}
	return accOf(pic)
}

func (t *Text) measure(maxW float32) (float32, float32) {
	face := t.face()
	lineH := lewimage.LineHeight(face)
	if maxW < 1 {
		maxW = unbounded
	}
	col := 0
	lines := 1
	maxCol := 0
	for _, r := range t.Value {
		if r == '\n' {
			if col > maxCol {
				maxCol = col
			}
			col = 0
			lines++
			continue
		}
		adv := lewimage.Advance(r, face)
		if col+adv > int(maxW) && col > 0 {
			if col > maxCol {
				maxCol = col
			}
			col = adv
			lines++
			continue
		}
		col += adv
	}
	if col > maxCol {
		maxCol = col
	}
	return float32(maxCol), float32(lines * lineH)
}

func (run textRun) stamp(dst *image.RGBA) {
	if dst == nil || run.box.Width < 1 || run.box.Height < 1 {
		return
	}
	face := lewimage.Use(run.face)
	x0 := int(run.box.X)
	maxX := x0 + int(run.box.Width)
	maxY := int(run.box.Y + run.box.Height)
	col := x0
	baseline := int(run.box.Y) + lewimage.Ascent(face)
	lineH := lewimage.LineHeight(face)
	i := 0
	for _, r := range run.body {
		adv := lewimage.Advance(r, face)
		if r == '\n' {
			if run.caret && i == run.cursor {
				run.drawCaret(dst, col, baseline, maxY)
			}
			col = x0
			baseline += lineH
			i++
			continue
		}
		if col+adv > maxX && col > x0 {
			col = x0
			baseline += lineH
		}
		if run.caret && i == run.cursor {
			run.drawCaret(dst, col, baseline, maxY)
		}
		if baseline <= maxY && r >= 32 {
			lewimage.Stamp{Dst: dst, X: col, Y: baseline, Text: string(r), Face: face}.Draw()
		}
		col += adv
		i++
	}
	if run.caret && i == run.cursor {
		run.drawCaret(dst, col, baseline, maxY)
	}
}

func (run textRun) drawCaret(dst *image.RGBA, x, baseline, maxY int) {
	face := lewimage.Use(run.face)
	top := baseline - lewimage.Ascent(face)
	if dst == nil || top < 0 || x < 0 || x >= dst.Bounds().Dx() {
		return
	}
	bottom := min(baseline+2, maxY, dst.Bounds().Dy())
	if bottom <= top {
		return
	}
	draw.Draw(dst, image.Rect(x, top, x+2, bottom), caretFill, image.Point{}, draw.Src)
}

func (t *Text) indexAt(pos image.Point) int {
	if t == nil {
		return 0
	}
	face := t.face()
	body := []rune(t.Value)
	lineH := lewimage.LineHeight(face)
	row := max(0, (pos.Y-int(t.origin.Y))/max(1, lineH))
	targetX := pos.X - int(t.origin.X)
	maxW := int(t.size.Width)
	r, x := 0, 0
	for i, ch := range body {
		if r > row {
			return i - 1
		}
		if ch == '\n' {
			if r == row {
				return i
			}
			r++
			x = 0
			continue
		}
		adv := lewimage.Advance(ch, face)
		if x+adv > maxW && x > 0 {
			r++
			x = 0
			if r > row {
				return i
			}
		}
		if r == row && x+adv/2 >= targetX {
			return i
		}
		x += adv
	}
	return len(body)
}
