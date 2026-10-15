package gui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCurvesEndpointsAndMonotonicity(t *testing.T) {
	curves := []Curve{
		Linear,
		Ease,
		EaseIn,
		EaseOut,
		EaseInOut,
		FastOutSlowIn,
		EaseOutCubic,
		EaseInOutCubic,
		EaseOutQuart,
	}
	for _, curve := range curves {
		assert.InDelta(t, 0, curve.Transform(0), 1e-4)
		assert.InDelta(t, 1, curve.Transform(1), 1e-4)
		mid := curve.Transform(0.5)
		assert.Greater(t, mid, float32(0.05))
		assert.Less(t, mid, float32(0.98))
	}
	assert.Greater(t, EaseOutBack.Transform(0.7), float32(1.0))
}

func TestLerpPrimitives(t *testing.T) {
	assert.InDelta(t, 50, Lerp(0, 100, 0.5), 1e-4)
	c := LerpRGB(RGB{0, 100, 200, 255}, RGB{200, 200, 0, 55}, 0.5)
	assert.Equal(t, RGB{100, 150, 100, 155}, c)
	off := LerpOffset(Offset{10, 20}, Offset{30, 60}, 0.25)
	assert.Equal(t, Offset{15, 30}, off)
	sz := LerpSize(Size{100, 200}, Size{200, 400}, 0.5)
	assert.Equal(t, Size{150, 300}, sz)
}

func TestTweenLifecycle(t *testing.T) {
	tween := NewTween(10)
	assert.False(t, tween.Animating())
	assert.Equal(t, float32(10), tween.Value())

	tween.To(110, 100*time.Millisecond, Linear)
	assert.True(t, tween.Animating())
	assert.True(t, tween.Step(50*time.Millisecond))
	assert.InDelta(t, 60, tween.Value(), 0.5)
	assert.False(t, tween.Step(50*time.Millisecond))
	assert.Equal(t, float32(110), tween.Value())
	assert.False(t, tween.Animating())
}

func TestSpringSettlesAtTarget(t *testing.T) {
	var spring Spring
	spring.Set(0)
	spring.Target = 240
	steps := 0
	for spring.Step(16 * time.Millisecond) {
		steps++
		require.Less(t, steps, 200)
	}
	assert.Equal(t, float32(240), spring.Value)
	assert.False(t, spring.Animating())
}

func TestOpacityAndSlideNodes(t *testing.T) {
	fill := RGB{200, 100, 50, 200}
	root := &Opacity{
		Value: 0.5,
		Child: &Slide{
			Offset:     Offset{X: 10, Y: 5},
			Fractional: Offset{X: 0.25, Y: 0},
			Child:      &Box{Width: 40, Height: 20, Fill: &fill, Child: &Text{Value: "hi", Ink: RGB{255, 255, 255, 255}}},
		},
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	_, err = picture.Render(root, Size{Width: 100, Height: 60})
	require.NoError(t, err)
	require.Len(t, picture.fills, 1)
	assert.InDelta(t, 35, picture.fills[0].X, 0.01)
	assert.InDelta(t, 5, picture.fills[0].Y, 0.01)
	assert.InDelta(t, 100, picture.fills[0].Alpha, 1)
	require.Len(t, picture.texts, 1)
	assert.InDelta(t, 128, picture.texts[0].ink.Alpha, 1)
}

func TestColumnViewScrollAndThumb(t *testing.T) {
	thumb := RGB{180, 180, 200, 160}
	itemFill := RGB{40, 50, 80, 255}
	items := make([]Node, 6)
	for i := range items {
		items[i] = &Box{Height: 30, Fill: &itemFill}
	}
	view := &ColumnView{
		Gap:      8,
		Padding:  EdgeInsets{Left: 4, Top: 4, Right: 4, Bottom: 4},
		Offset:   20,
		Thumb:    &thumb,
		Children: items,
	}
	picture, err := NewPicture()
	require.NoError(t, err)
	picture.recordOnly = true
	_, err = picture.Render(view, Size{Width: 120, Height: 80})
	require.NoError(t, err)
	assert.Equal(t, float32(6*30+5*8+8), view.ContentHeight())
	assert.Equal(t, float32(6*30+5*8+8-80), view.MaxOffset())
	require.Len(t, picture.fills, 7)
	assert.Equal(t, float32(4-20), picture.fills[0].Y)
}

