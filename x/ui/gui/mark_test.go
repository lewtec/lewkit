package gui

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterViewRecordsMarksInPaintOrder(t *testing.T) {
	counter, err := NewCounter()
	require.NoError(t, err)
	root := counter.View()
	root.Layout(Tight(400, 200))
	picture, err := NewPicture()
	require.NoError(t, err)
	root.Paint(Offset{}, Rect{0, 0, 400, 200}, picture)

	var kinds []MarkKind
	for _, mark := range picture.Marks() {
		kinds = append(kinds, mark.Kind)
	}
	assert.Equal(t, []MarkKind{
		MarkFill, MarkFill, MarkText,
		MarkText,
		MarkFill, MarkText,
	}, kinds)

	marks := picture.Marks()
	assert.Equal(t, "-", marks[2].Text)
	assert.Equal(t, "0", marks[3].Text)
	assert.Equal(t, "+", marks[5].Text)
	_, ink := Palette(counter.mode)
	assert.Equal(t, ink, marks[2].Color)
}

func TestImageAndBackdropAreMarks(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	src := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pixels, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	picture.Backdrop(pixels)
	picture.Image(src, Rect{1, 2, 3, 4}, Rect{1, 2, 3, 4}, 2)
	marks := picture.Marks()
	require.Len(t, marks, 2)
	assert.Equal(t, MarkBackdrop, marks[0].Kind)
	assert.Same(t, pixels, marks[0].Pixels)
	assert.Equal(t, MarkImage, marks[1].Kind)
	assert.Equal(t, float32(2), marks[1].Radius)
	assert.Equal(t, src, marks[1].Src)
}
