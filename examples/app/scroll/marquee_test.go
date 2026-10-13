package main

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapShift(t *testing.T) {
	assert.InDelta(t, 10, wrapShift(70+40, 100), 0.01)
	assert.InDelta(t, 90, wrapShift(10-20, 100), 0.01)
}

func TestMarqueeTreeReused(t *testing.T) {
	marquee, err := NewMarquee()
	require.NoError(t, err)
	marquee.size = image.Pt(80, 200)
	first := marquee.View()
	marquee.offset = 40
	second := marquee.View()
	require.Same(t, first, second)
	marquee.size = image.Pt(80, 400)
	later := marquee.View()
	require.NotSame(t, first, later)
}

func TestMarqueeLoops(t *testing.T) {
	marquee, err := NewMarquee()
	require.NoError(t, err)
	marquee.size = image.Pt(80, 120)
	before := barYs(t, marquee)
	marquee.offset = 160
	after := barYs(t, marquee)
	require.NotEmpty(t, before)
	require.Equal(t, len(before), len(after))
	assert.Greater(t, after[0], before[0])
}

func TestMarqueeFillsViewport(t *testing.T) {
	const width, height = 80, 400
	top, bot := float32(marqueePad), float32(height-marqueePad)
	for _, delta := range []float32{0, 32, 96, 320} {
		marquee, err := NewMarquee()
		require.NoError(t, err)
		marquee.size = image.Pt(width, height)
		marquee.offset = delta
		bars := marqueeBars(t, marquee, width, height)
		require.NotEmpty(t, bars, "delta %v", delta)
		empty, maxEmpty := 0, 0
		for y := top + 1; y < bot-1; y++ {
			covered := false
			for _, fill := range bars {
				if y >= fill.Y && y < fill.Y+fill.Height {
					covered = true
					break
				}
			}
			if covered {
				empty = 0
				continue
			}
			empty++
			if empty > maxEmpty {
				maxEmpty = empty
			}
		}
		assert.LessOrEqual(t, maxEmpty, marqueeGap+1, "delta %v", delta)
	}
}

func TestMarqueeScrollThenTick(t *testing.T) {
	marquee, err := NewMarquee()
	require.NoError(t, err)
	marquee.size = image.Pt(80, 200)
	marquee.lastTick = 100 * time.Millisecond
	next, cmd := marquee.Update(window.Scroll{Delta: image.Pt(0, -20)})
	require.Same(t, marquee, next)
	require.Nil(t, cmd)
	assert.InDelta(t, marquee.wrap(-20), marquee.offset, 0.01)
	next, cmd = marquee.Update(gui.TickMsg{Elapsed: 150 * time.Millisecond, Size: marquee.size})
	require.Same(t, marquee, next)
	require.NotNil(t, cmd)
	assert.InDelta(t, marquee.wrap(-20+4), marquee.offset, 0.01)
}

func TestMarqueeResizeClearsDrag(t *testing.T) {
	marquee, err := NewMarquee()
	require.NoError(t, err)
	marquee.size = image.Pt(80, 200)
	marquee.lastTick = 100 * time.Millisecond
	_, _ = marquee.Update(window.Pointer{Pos: image.Pt(10, 50), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.True(t, marquee.dragging)
	offset := marquee.offset
	_, cmd := marquee.Update(window.Resize{Size: image.Pt(80, 240)})
	require.Nil(t, cmd)
	assert.False(t, marquee.dragging)
	next, cmd := marquee.Update(gui.TickMsg{Elapsed: 200 * time.Millisecond, Period: time.Second / 60, Size: image.Pt(80, 240)})
	require.Same(t, marquee, next)
	require.NotNil(t, cmd)
	assert.Greater(t, marquee.offset, offset)
}

func TestMarqueeDrag(t *testing.T) {
	marquee, err := NewMarquee()
	require.NoError(t, err)
	marquee.size = image.Pt(80, 200)
	next, cmd := marquee.Update(window.Pointer{Pos: image.Pt(10, 50), Button: 1, Pressed: true, Buttons: window.ButtonLeft})
	require.Same(t, marquee, next)
	require.Nil(t, cmd)
	next, cmd = marquee.Update(window.Pointer{Pos: image.Pt(10, 90), Buttons: window.ButtonLeft})
	require.Same(t, marquee, next)
	require.Nil(t, cmd)
	assert.InDelta(t, 40, marquee.offset, 0.01)
	assert.True(t, marquee.dragging)
	next, cmd = marquee.Update(window.Pointer{Pos: image.Pt(10, 90), Button: 1, Pressed: false})
	require.Same(t, marquee, next)
	require.Nil(t, cmd)
	assert.False(t, marquee.dragging)
}

func barYs(t *testing.T, marquee *Marquee) []float32 {
	t.Helper()
	var ys []float32
	for _, fill := range marqueeBars(t, marquee, marquee.size.X, marquee.size.Y) {
		ys = append(ys, fill.Y)
	}
	return ys
}

func marqueeBars(t *testing.T, marquee *Marquee, width, height int) []gui.Rect {
	t.Helper()
	root := marquee.View()
	root.Layout(gui.Tight(float32(width), float32(height)))
	picture, err := gui.NewPicture()
	require.NoError(t, err)
	root.Paint(gui.Offset{}, gui.Rect{0, 0, float32(width), float32(height)}, picture)
	var bars []gui.Rect
	for _, mark := range picture.Marks() {
		fill := mark.Box
		if mark.Kind == gui.MarkFill && mark.Color.Alpha > 0 && fill.Height <= marqueeItemHeight+1 && fill.Width < float32(width) {
			bars = append(bars, fill)
		}
	}
	return bars
}
