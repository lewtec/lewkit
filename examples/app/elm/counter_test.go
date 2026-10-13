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
		gui.MarkFill, gui.MarkFill, gui.MarkText,
		gui.MarkText,
		gui.MarkFill, gui.MarkText,
	}, kinds)

	marks := picture.Marks()
	assert.Equal(t, "-", marks[2].Text)
	assert.Equal(t, "0", marks[3].Text)
	assert.Equal(t, "+", marks[5].Text)
	_, ink := gui.Palette(counter.mode)
	assert.Equal(t, ink, marks[2].Color)
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
