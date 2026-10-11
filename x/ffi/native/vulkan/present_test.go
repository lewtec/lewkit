package vulkan

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPresentExtent(t *testing.T) {
	w, h := presentExtent(2400, 1080, transformIdent)
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	w, h = presentExtent(2400, 1080, transform90)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
}

func TestChooseSwapchainKeepsTheLandscapeExtent(t *testing.T) {
	// The phone is sideways. The surface already reports the landscape
	// buffer. currentTransform is 90°. Swapping that extent again is the
	// create that Vulkan rejects.
	caps := surfaceCaps{
		currentW: 2400, currentH: 1080,
		currentTrans: transform90,
		transforms:   transformIdent | transform90 | transform180 | transform270,
	}
	w, h, turn := chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	assert.Equal(t, uint32(transformIdent), turn)

	w, h, turn = chooseSwapchain(1080, 2400, surfaceCaps{
		currentW: 1080, currentH: 2400, currentTrans: transformIdent, transforms: transformIdent,
	})
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transformIdent), turn)
}

func TestChooseSwapchainStoresATransposedBuffer(t *testing.T) {
	caps := surfaceCaps{
		currentW: 1080, currentH: 2400,
		currentTrans: transform90,
		transforms:   transformIdent | transform90,
	}
	w, h, turn := chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform90), turn)
	gotW, gotH := presentExtent(2400, 1080, turn)
	assert.Equal(t, w, gotW)
	assert.Equal(t, h, gotH)

	caps.currentTrans = transform270
	_, _, turn = chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, uint32(transform270), turn)
}

func TestChooseSwapchainLeavesAnUndefinedExtentUpright(t *testing.T) {
	caps := surfaceCaps{
		currentW: extentUnset, currentH: extentUnset,
		minW: 1, minH: 1, maxW: 4096, maxH: 4096,
		currentTrans: transform90,
		transforms:   transformIdent | transform90,
	}
	w, h, turn := chooseSwapchain(800, 400, caps)
	assert.Equal(t, 800, w)
	assert.Equal(t, 400, h)
	assert.Equal(t, uint32(transformIdent), turn)
}

func TestNoteResultKeepsALostSurfaceRetryable(t *testing.T) {
	var screen Screen
	err := screen.noteResult(outOfDate)
	assert.ErrorIs(t, err, ErrLost)
	assert.True(t, screen.stale)
	assert.ErrorIs(t, screen.noteResult(surfaceLost), ErrLost)
	assert.ErrorIs(t, screen.noteResult(nativeWindowUse), ErrLost)
	screen.stale = false
	assert.NoError(t, screen.noteResult(suboptimal))
	assert.True(t, screen.stale)
	assert.NoError(t, screen.noteResult(0))
	assert.True(t, surfaceGone(fmt.Errorf("swapchain: %w format=37", Result(outOfDate))))
	assert.False(t, surfaceGone(Result(-3)))
}

func TestPresentSourceRotatesUprightPixel(t *testing.T) {
	const width, height = 8, 4
	sx, sy := presentSource(transform90, width, height, 0, 0)
	assert.Equal(t, 0, sx)
	assert.Equal(t, height-1, sy)
	sx, sy = presentSource(transform270, width, height, 0, 0)
	assert.Equal(t, width-1, sx)
	assert.Equal(t, 0, sy)
	sx, sy = presentSource(transform180, width, height, 1, 2)
	assert.Equal(t, width-2, sx)
	assert.Equal(t, height-3, sy)
}
