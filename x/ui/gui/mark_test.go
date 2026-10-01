package gui

import (
	"image"
	"slices"
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

func TestLowerProjectsMarks(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1, 1))
	first, err := ndarray.New([]float32{1, 0, 0, 1}, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	second, err := ndarray.New([]float32{9, 8, 7, 6}, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	marks := []Mark{
		{Kind: MarkBackdrop, Pixels: first},
		{Kind: MarkFill, Box: Rect{1, 2, 3, 4}, Clip: Rect{0, 0, 9, 9}, Radius: 2, Color: RGB{4, 5, 6, 255}},
		{Kind: MarkText, Box: Rect{1, 1, 8, 8}, Text: "ab", Color: RGB{255, 255, 255, 255}, Cursor: 1, Caret: true},
		{Kind: MarkImage, Src: src, Box: Rect{1, 1, 2, 2}, Clip: Rect{1, 1, 2, 2}, Radius: 1},
		{Kind: MarkBackdrop, Pixels: nil},
		{Kind: MarkBackdrop, Pixels: second},
		{Kind: MarkImage},
	}
	got := lower(slices.Values(marks))
	require.Len(t, got.fills, 1)
	assert.Equal(t, float32(1), got.fills[0].X)
	assert.Equal(t, float32(4), got.fills[0].Red)
	assert.Equal(t, float32(2), got.fills[0].Radius)
	assert.Equal(t, float32(9), got.fills[0].ClipWidth)
	require.Len(t, got.texts, 1)
	assert.Equal(t, []rune("ab"), got.texts[0].body)
	assert.Equal(t, 1, got.texts[0].cursor)
	assert.True(t, got.texts[0].caret)
	assert.Equal(t, RGB{255, 255, 255, 255}, got.texts[0].ink)
	require.Len(t, got.images, 1)
	assert.Same(t, src, got.images[0].src)
	assert.Equal(t, float32(1), got.images[0].radius)
	assert.Same(t, second, got.backdrop)
}

func TestPaintMarksStreamsTheView(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Stack{Children: []Node{
		&Box{Fill: &RGB{1, 2, 3, 255}},
		&Box{Fill: &RGB{4, 5, 6, 255}},
	}}
	root.Layout(Tight(8, 8))
	var streamed []Mark
	for mark := range picture.paintMarks(root, Size{8, 8}) {
		streamed = append(streamed, mark)
	}
	assert.Equal(t, streamed, picture.Marks())
	require.Len(t, streamed, 2)
	assert.Equal(t, uint8(1), streamed[0].Color.Red)
	assert.Equal(t, uint8(4), streamed[1].Color.Red)
}

func TestPaintMarksStops(t *testing.T) {
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Stack{Children: []Node{
		&Box{Fill: &RGB{1, 0, 0, 255}},
		&Box{Fill: &RGB{2, 0, 0, 255}},
		&Box{Fill: &RGB{3, 0, 0, 255}},
	}}
	root.Layout(Tight(10, 10))
	var seen int
	for range picture.paintMarks(root, Size{10, 10}) {
		seen++
		if seen == 1 {
			break
		}
	}
	assert.Equal(t, 1, seen)
	marks := picture.Marks()
	require.Len(t, marks, 1)
	assert.Equal(t, uint8(1), marks[0].Color.Red)
}

func TestRenderLowersMarks(t *testing.T) {
	pixels, err := ndarray.New([]float32{1, 2, 3, 4}, ndarray.Shape{1, 1, 4})
	require.NoError(t, err)
	picture, err := NewPicture()
	require.NoError(t, err)
	root := &Stack{Children: []Node{
		&Raster{Pixels: pixels},
		&Box{Width: 4, Height: 4, Fill: &RGB{9, 8, 7, 255}},
		&Box{Width: 2, Height: 2, Fill: &RGB{1, 0, 0, 128}},
	}}
	_, err = picture.Render(root, Size{16, 16})
	require.NoError(t, err)
	got := lower(slices.Values(picture.Marks()))
	assert.Equal(t, got.fills, picture.fills)
	assert.Empty(t, picture.texts)
	assert.Empty(t, picture.images)
	assert.Same(t, pixels, picture.raster)
	require.Len(t, picture.marks, 3)
	require.Len(t, picture.fills, 2)
}
