package main

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterFollowsScheme(t *testing.T) {
	counter, err := NewCounter()
	require.NoError(t, err)
	dark := counter.View().(*gui.Box)
	require.Equal(t, gui.RGB{28, 28, 34, 255}, *dark.Fill)
	next, cmd := counter.Update(gui.ModeMsg{Mode: daynight.Light})
	require.Nil(t, cmd)
	light := next.View().(*gui.Box)
	require.Equal(t, gui.RGB{246, 246, 244, 255}, *light.Fill)
	text := light.Child.(*gui.Flex).Children[2].Child.(*gui.Box).Child.(*gui.Text)
	require.Equal(t, gui.RGB{28, 28, 34, 255}, text.Ink)
}

func TestCounterViewRecordsMarksInPaintOrder(t *testing.T) {
	counter, err := NewCounter()
	require.NoError(t, err)
	root := counter.View()
	root.Layout(gui.Tight(400, 200))
	picture, err := gui.NewPicture()
	require.NoError(t, err)
	root.Paint(gui.Offset{}, gui.Rect{0, 0, 400, 200}, picture)

	var kinds []gui.MarkKind
	for _, mark := range picture.Marks() {
		kinds = append(kinds, mark.Kind)
	}
	assert.Equal(t, []gui.MarkKind{
		gui.MarkFill, gui.MarkFill, gui.MarkImage,
		gui.MarkText,
		gui.MarkFill, gui.MarkImage,
	}, kinds)

	marks := picture.Marks()
	assert.Equal(t, "0", marks[3].Text)
	_, ink := gui.Palette(counter.mode)
	assert.True(t, paintedInk(marks[2].Src, ink))
	assert.True(t, paintedInk(marks[5].Src, ink))
}

func paintedInk(src image.Image, ink gui.RGB) bool {
	if src == nil {
		return false
	}
	bounds := src.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			red, green, blue, alpha := src.At(x, y).RGBA()
			if alpha < 0xc000 {
				continue
			}
			if uint8(red>>8) == ink.Red && uint8(green>>8) == ink.Green && uint8(blue>>8) == ink.Blue {
				return true
			}
		}
	}
	return false
}

func TestCounterButtons(t *testing.T) {
	counter, err := NewCounter()
	require.NoError(t, err)
	counter.size = image.Pt(400, 200)
	root := counter.View()
	require.NotNil(t, root)
	root.Layout(gui.Tight(400, 200))
	picture, err := gui.NewPicture()
	require.NoError(t, err)
	root.Paint(gui.Offset{}, gui.Rect{0, 0, 400, 200}, picture)
	require.NotNil(t, counter.plus)
	next, cmd := counter.Update(window.Pointer{Pos: counter.plus.Center(), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, counter, next)
	require.Nil(t, cmd)
	assert.Equal(t, 1, counter.count)
	next, cmd = counter.Update(window.Pointer{Pos: counter.minus.Center(), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, counter, next)
	require.Nil(t, cmd)
	assert.Equal(t, 0, counter.count)
}
