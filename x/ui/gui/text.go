package gui

import (
	"image"
	"image/color"
	"image/draw"

	lewimage "github.com/lewtec/lewkit/x/image"
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
	w, h := measureText([]rune(t.Value), t.face(), c.MaxWidth)
	t.size = c.Constrain(Size{Width: w, Height: h})
	return t.size
}

func (t *Text) Paint(origin Offset, clip Rect, paint *painter) {
	if t == nil || paint == nil {
		return
	}
	t.origin = origin
	box := Rect{origin.X, origin.Y, t.size.Width, t.size.Height}.Intersect(clip)
	paint.texts = append(paint.texts, textRun{
		box:    box,
		body:   []rune(t.Value),
		face:   t.face(),
		cursor: t.Cursor,
		caret:  t.Caret,
	})
}

func measureText(body []rune, face font.Face, maxW float32) (float32, float32) {
	lineH := lewimage.LineHeight(face)
	if maxW < 1 {
		maxW = unbounded
	}
	col := 0
	lines := 1
	maxCol := 0
	for _, r := range body {
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

func stampRun(dst *image.RGBA, run textRun) {
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
				drawCaret(dst, col, baseline, maxY, face)
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
			drawCaret(dst, col, baseline, maxY, face)
		}
		if baseline <= maxY && r >= 32 {
			lewimage.Stamp{Dst: dst, X: col, Y: baseline, Text: string(r), Face: face}.Draw()
		}
		col += adv
		i++
	}
	if run.caret && i == run.cursor {
		drawCaret(dst, col, baseline, maxY, face)
	}
}

func drawCaret(dst *image.RGBA, x, baseline, maxY int, face font.Face) {
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

func textIndex(box Rect, pos image.Point, body []rune, face font.Face) int {
	face = lewimage.Use(face)
	lineH := lewimage.LineHeight(face)
	row := max(0, (pos.Y-int(box.Y))/max(1, lineH))
	targetX := pos.X - int(box.X)
	maxW := int(box.Width)
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
