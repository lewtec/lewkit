package gui

import (
	"image"
	"testing"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDragTapStaysInsideSlop(t *testing.T) {
	var drag Drag
	_, ok := drag.Step(window.Pointer{Pos: image.Pt(10, 10), Button: 1, Pressed: true, Buttons: window.ButtonLeft}, 8)
	require.False(t, ok)
	_, ok = drag.Step(window.Pointer{Pos: image.Pt(14, 16), Buttons: window.ButtonLeft}, 8)
	require.False(t, ok)
	step, ok := drag.Step(window.Pointer{Pos: image.Pt(14, 16), Button: 1}, 8)
	require.True(t, ok)
	assert.True(t, step.Tap)
	assert.Equal(t, image.Pt(14, 16), step.At)
}

func TestDragReportsFingerDelta(t *testing.T) {
	var drag Drag
	_, ok := drag.Step(window.Pointer{Pos: image.Pt(40, 200), Button: 1, Pressed: true, Buttons: window.ButtonLeft}, 8)
	require.False(t, ok)
	step, ok := drag.Step(window.Pointer{Pos: image.Pt(40, 120), Buttons: window.ButtonLeft}, 8)
	require.True(t, ok)
	assert.False(t, step.Tap)
	assert.Equal(t, image.Pt(0, 80), step.Delta)
	_, ok = drag.Step(window.Pointer{Pos: image.Pt(40, 120), Button: 1}, 8)
	assert.False(t, ok)
}

func TestSlopFollowsDensity(t *testing.T) {
	assert.Equal(t, 8, Slop(0))
	assert.Equal(t, 20, Slop(2.5))
	assert.InDelta(t, float32(40), Logical(2.5, 100), 0.01)
	assert.Equal(t, float32(100), Logical(0, 100))
}
