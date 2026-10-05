package gui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectEmptyIsWindow(t *testing.T) {
	window := Rect{10, 20, 100, 80}
	assert.Equal(t, window, Detect(window, nil))
	assert.Equal(t, window, Detect(window, []Rect{{400, 400, 8, 8}, {0, 0, 0, 12}}))
	assert.Equal(t, Rect{}, Detect(Rect{0, 0, 0, 40}, nil))
}

func TestDetectNavbarAndNotch(t *testing.T) {
	window := Rect{0, 0, 100, 80}
	notch := Rect{30, 0, 40, 20}
	navbar := Rect{0, 64, 100, 16}
	assert.Equal(t, Rect{0, 20, 100, 44}, Detect(window, []Rect{notch, navbar}))
}

func TestDetectSideCutout(t *testing.T) {
	window := Rect{0, 0, 100, 100}
	// A tall left cutout leaves more room beside it than below it.
	assert.Equal(t, Rect{30, 0, 70, 100}, Detect(window, []Rect{{0, 0, 30, 80}}))
	// A shallow top notch leaves more room below it than beside it.
	assert.Equal(t, Rect{0, 20, 100, 80}, Detect(window, []Rect{{30, 0, 40, 20}}))
}

func TestDetectCoveredWindow(t *testing.T) {
	window := Rect{0, 0, 50, 40}
	assert.Equal(t, Rect{}, Detect(window, []Rect{{-10, -10, 80, 80}}))
}

func TestDetectOverlappingBars(t *testing.T) {
	window := Rect{0, 0, 100, 100}
	shallow := Rect{0, 0, 100, 10}
	deeper := Rect{0, 0, 100, 25}
	assert.Equal(t, Rect{0, 25, 100, 75}, Detect(window, []Rect{shallow, deeper}))
}

func TestDetectEqualAreaPrefersWider(t *testing.T) {
	// A square corner cutout ties the band below it with the column beside it.
	// The wider band is the suggestion.
	assert.Equal(t, Rect{0, 50, 100, 50}, Detect(Rect{0, 0, 100, 100}, []Rect{{0, 0, 50, 50}}))
}

func TestHintPlacesChildInShownBox(t *testing.T) {
	child := &Box{Fill: &RGB{255, 0, 0, 255}}
	hint := &Hint{
		Dead:  []Rect{{0, 0, 100, 12}, {0, 70, 100, 10}},
		Child: child,
	}
	assert.Equal(t, Size{100, 80}, hint.Layout(Tight(100, 80)))
	assert.Equal(t, Rect{0, 12, 100, 58}, hint.Shown())
	picture, err := NewPicture()
	require.NoError(t, err)
	hint.Paint(Offset{3, 4}, Rect{3, 4, 100, 80}, picture)
	require.Len(t, picture.fills, 1)
	fill := picture.fills[0]
	assert.Equal(t, float32(3), fill.X)
	assert.Equal(t, float32(16), fill.Y)
	assert.Equal(t, float32(100), fill.Width)
	assert.Equal(t, float32(58), fill.Height)
	assert.Equal(t, float32(3), fill.ClipX)
	assert.Equal(t, float32(16), fill.ClipY)
	assert.Equal(t, float32(100), fill.ClipWidth)
	assert.Equal(t, float32(58), fill.ClipHeight)
}

func TestHintClipsDeadZone(t *testing.T) {
	hint := &Hint{
		Dead: []Rect{{0, 0, 40, 8}},
		Child: &Stack{Children: []Node{
			&Positioned{Y: -8, Child: &Box{Width: 40, Height: 24, Fill: &RGB{255, 0, 0, 255}}},
		}},
	}
	pixels := raster(t, hint, 40, 24)
	assert.Equal(t, uint8(0), at(pixels, 40, 4, 2).R)
	assert.InDelta(t, 255, at(pixels, 40, 4, 12).R, 2)
	assert.Equal(t, uint8(0), at(pixels, 40, 4, 12).G)
}

func TestHintCoveredDrawsNothing(t *testing.T) {
	hint := &Hint{Dead: []Rect{{0, 0, 8, 8}}, Child: &Box{Fill: &RGB{255, 0, 0, 255}}}
	pixels := raster(t, hint, 8, 8)
	assert.Equal(t, uint8(0), at(pixels, 8, 3, 3).R)
	assert.Equal(t, Rect{}, hint.Shown())
}

func TestHintWithoutChild(t *testing.T) {
	hint := &Hint{Dead: []Rect{{0, 0, 10, 2}}}
	assert.Equal(t, Size{10, 10}, hint.Layout(Tight(10, 10)))
	assert.Equal(t, Rect{0, 2, 10, 8}, hint.Shown())
	var bare *Hint
	assert.Equal(t, Rect{}, bare.Shown())
	assert.Equal(t, Size{}, bare.Layout(Tight(10, 10)))
}
