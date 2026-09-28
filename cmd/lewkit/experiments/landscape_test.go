package experiments

import (
	"image"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTriangleLandscapeReflow(t *testing.T) {
	const width, height = 800, 360
	model, err := newSpinModel(triangleTurnsPerSecond, 360, 800, triangleDynamic)
	require.NoError(t, err)

	portrait := image.Pt(360, 800)
	_, _ = model.Update(window.Resize{Size: portrait})
	_, _ = model.Update(gui.TickMsg{Elapsed: time.Second, Size: portrait, Period: time.Second / 60})
	assert.Equal(t, portrait, model.size)

	landscape := image.Pt(width, height)
	_, _ = model.Update(window.Resize{Size: landscape})
	_, _ = model.Update(gui.TickMsg{Elapsed: time.Second + time.Second/60, Size: landscape, Period: time.Second / 60})
	assert.Equal(t, landscape, model.size)

	picture, err := gui.NewPicture()
	require.NoError(t, err)
	view := model.View()
	pixels, err := picture.Render(view, gui.Size{Width: width, Height: height})
	require.NoError(t, err)
	require.NotNil(t, pixels)
	require.True(t, model.minus.Contains(image.Pt(312, 296)), "minus sits on the landscape bottom")
	require.True(t, model.plus.Contains(image.Pt(488, 296)), "plus sits on the landscape bottom")
	require.False(t, model.minus.Contains(image.Pt(312, 40)))

	out := make([]uint8, width*height*4)
	require.NoError(t, pixels.Eval(t.Context(), ndarray.CPU, out))
	center := sample(out, width, width/2, height/2)
	assert.Greater(t, int(center[0])+int(center[1])+int(center[2]), 40, "triangle center should be on screen")
	edge := sample(out, width, width-8, height/2)
	assert.Less(t, int(edge[0])+int(edge[1])+int(edge[2]), 30, "landscape side should be outside the triangle")
}

func sample(pixels []uint8, width, x, y int) []uint8 {
	i := (y*width + x) * 4
	return pixels[i : i+4]
}
