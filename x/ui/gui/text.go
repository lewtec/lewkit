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

// Text is a layout node. Nil Face uses [lewimage.Face].
// Cursor -1 hides the caret. Zero Ink keeps the white default.
type Text struct {
	Value  string
	Face   font.Face
	Cursor int
	Caret  bool
	// Ink is the glyph color. Zero keeps the white default.
	Ink Color

	origin Offset
	size   Size
}

func (text *Text) face() font.Face {
	if text == nil {
		return lewimage.Face()
	}
	return lewimage.Use(text.Face)
}

func (text *Text) Layout(constraints BoxConstraints) Size {
	if text == nil {
		return Size{}
	}
	width, height := text.measure(constraints.MaxWidth)
	text.size = constraints.Constrain(Size{Width: width, Height: height})
	return text.size
}

func (text *Text) Paint(origin Offset, clip Rect, picture *Picture) *ndarray.Tensor[float32] {
	if text == nil {
		return accumulatorOf(picture)
	}
	text.origin = origin
	if picture != nil {
		full := Rect{origin.X, origin.Y, text.size.Width, text.size.Height}
		visible := full.Intersect(clip)
		if visible.Width < 1 || visible.Height < 1 {
			return accumulatorOf(picture)
		}
		picture.glyph(textRun{
			box:    full,
			clip:   visible,
			body:   []rune(text.Value),
			face:   text.face(),
			cursor: text.Cursor,
			caret:  text.Caret,
			ink:    text.Ink,
		})
	}
	return accumulatorOf(picture)
}

func (text *Text) measure(maxWidth float32) (float32, float32) {
	face := text.face()
	lineHeight := lewimage.LineHeight(face)
	if maxWidth < 1 {
		maxWidth = unbounded
	}
	column := 0
	lines := 1
	maxColumn := 0
	for _, character := range text.Value {
		if character == '\n' {
			if column > maxColumn {
				maxColumn = column
			}
			column = 0
			lines++
			continue
		}
		advance := lewimage.Advance(character, face)
		if column+advance > int(maxWidth) && column > 0 {
			if column > maxColumn {
				maxColumn = column
			}
			column = advance
			lines++
			continue
		}
		column += advance
	}
	if column > maxColumn {
		maxColumn = column
	}
	return float32(maxColumn), float32(lines * lineHeight)
}

func (run textRun) stamp(destination *image.RGBA) {
	if destination == nil || run.box.Width < 1 || run.box.Height < 1 {
		return
	}
	if clipped := run.destination(destination); clipped != nil {
		destination = clipped
	}
	face := lewimage.Use(run.face)
	originX := int(run.box.X)
	maxX := originX + int(run.box.Width)
	maxY := int(run.box.Y + run.box.Height)
	column := originX
	baseline := int(run.box.Y) + lewimage.Ascent(face)
	lineHeight := lewimage.LineHeight(face)
	index := 0
	for _, character := range run.body {
		advance := lewimage.Advance(character, face)
		if character == '\n' {
			if run.caret && index == run.cursor {
				run.drawCaret(destination, column, baseline, maxY)
			}
			column = originX
			baseline += lineHeight
			index++
			continue
		}
		if column+advance > maxX && column > originX {
			column = originX
			baseline += lineHeight
		}
		if run.caret && index == run.cursor {
			run.drawCaret(destination, column, baseline, maxY)
		}
		if baseline <= maxY && character >= 32 {
			lewimage.Stamp{Dst: destination, X: column, Y: baseline, Text: string(character), Face: face, Src: run.inkImage()}.Draw()
		}
		column += advance
		index++
	}
	if run.caret && index == run.cursor {
		run.drawCaret(destination, column, baseline, maxY)
	}
}

func (run textRun) destination(frame *image.RGBA) *image.RGBA {
	if run.clip.Width < 1 || run.clip.Height < 1 {
		return frame
	}
	rect := image.Rect(int(run.clip.X), int(run.clip.Y), int(run.clip.X+run.clip.Width), int(run.clip.Y+run.clip.Height))
	sub, ok := frame.SubImage(rect).(*image.RGBA)
	if !ok {
		return frame
	}
	return sub
}

func (run textRun) inkImage() image.Image {
	if run.ink.Alpha == 0 {
		return nil
	}
	return &image.Uniform{C: color.RGBA{R: run.ink.Red, G: run.ink.Green, B: run.ink.Blue, A: run.ink.Alpha}}
}

func (run textRun) drawCaret(destination *image.RGBA, x, baseline, maxY int) {
	face := lewimage.Use(run.face)
	top := baseline - lewimage.Ascent(face)
	if destination == nil || top < 0 || x < 0 || x >= destination.Bounds().Dx() {
		return
	}
	bottom := min(baseline+2, maxY, destination.Bounds().Dy())
	if bottom <= top {
		return
	}
	draw.Draw(destination, image.Rect(x, top, x+2, bottom), caretFill, image.Point{}, draw.Src)
}

func (text *Text) indexAt(position image.Point) int {
	if text == nil {
		return 0
	}
	face := text.face()
	body := []rune(text.Value)
	lineHeight := lewimage.LineHeight(face)
	row := max(0, (position.Y-int(text.origin.Y))/max(1, lineHeight))
	targetX := position.X - int(text.origin.X)
	maxWidth := int(text.size.Width)
	rowIndex, x := 0, 0
	for index, character := range body {
		if rowIndex > row {
			return index - 1
		}
		if character == '\n' {
			if rowIndex == row {
				return index
			}
			rowIndex++
			x = 0
			continue
		}
		advance := lewimage.Advance(character, face)
		if x+advance > maxWidth && x > 0 {
			rowIndex++
			x = 0
			if rowIndex > row {
				return index
			}
		}
		if rowIndex == row && x+advance/2 >= targetX {
			return index
		}
		x += advance
	}
	return len(body)
}
