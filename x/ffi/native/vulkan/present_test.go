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
	// buffer. currentTransform is 90°. The swapchain declares that
	// transform so present is not suboptimal, and the picture stays upright.
	caps := surfaceCaps{
		currentW: 2400, currentH: 1080,
		currentTrans: transform90,
		transforms:   transformIdent | transform90 | transform180 | transform270,
	}
	w, h, pre, content := chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transformIdent), content)

	w, h, pre, content = chooseSwapchain(1080, 2400, surfaceCaps{
		currentW: 1080, currentH: 2400, currentTrans: transformIdent, transforms: transformIdent,
	})
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transformIdent), pre)
	assert.Equal(t, uint32(transformIdent), content)
}

func TestChooseSwapchainStoresATransposedBuffer(t *testing.T) {
	caps := surfaceCaps{
		currentW: 1080, currentH: 2400,
		currentTrans: transform90,
		transforms:   transformIdent | transform90,
	}
	w, h, pre, content := chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transform90), content)
	gotW, gotH := presentExtent(2400, 1080, content)
	assert.Equal(t, w, gotW)
	assert.Equal(t, h, gotH)

	caps.currentTrans = transform270
	caps.transforms |= transform270
	_, _, pre, content = chooseSwapchain(2400, 1080, caps)
	assert.Equal(t, uint32(transform270), pre)
	assert.Equal(t, uint32(transform270), content)
}

func TestChooseSwapchainLeavesAnUndefinedExtentUpright(t *testing.T) {
	caps := surfaceCaps{
		currentW: extentUnset, currentH: extentUnset,
		minW: 1, minH: 1, maxW: 4096, maxH: 4096,
		currentTrans: transform90,
		transforms:   transformIdent | transform90,
	}
	w, h, pre, content := chooseSwapchain(800, 400, caps)
	assert.Equal(t, 800, w)
	assert.Equal(t, 400, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transformIdent), content)
}

func TestWaitIdleAfterPresentIsNotClosed(t *testing.T) {
	var waited, reset bool
	d := &Device{dev: 1, pending: true, fence: 7}
	d.api.waitForFences = func(device uintptr, count uint32, fences *uint64, waitAll uint32, timeout uint64) int32 {
		waited = true
		if device != 1 || count != 1 || fences == nil || *fences != 7 || waitAll != 1 || timeout != ^uint64(0) {
			return -1
		}
		return 0
	}
	d.api.resetFences = func(device uintptr, count uint32, fences *uint64) int32 {
		reset = true
		if fences == nil || *fences != 7 {
			return -1
		}
		return 0
	}
	assert.NoError(t, d.WaitIdle())
	assert.True(t, waited)
	assert.True(t, reset)
	assert.False(t, d.pending)

	// No fence loader still must not report the device closed.
	bare := &Device{dev: 1, pending: true}
	assert.NoError(t, bare.WaitIdle())
	assert.False(t, bare.pending)
}

func TestEnsureSemsWithoutImagesIsLost(t *testing.T) {
	screen := &Screen{d: &Device{dev: 1}}
	assert.ErrorIs(t, screen.ensureSems(), ErrLost)
	screen.swap = 1
	screen.images = []uint64{1}
	assert.ErrorIs(t, screen.ensureSems(), ErrClosed)
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

func TestOutputReportsSwapchainExtent(t *testing.T) {
	screen := &Screen{width: 1080, height: 2400, outW: 2400, outH: 1080}
	w, h := screen.Output()
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	upright := &Screen{width: 800, height: 400}
	w, h = upright.Output()
	assert.Equal(t, 800, w)
	assert.Equal(t, 400, h)
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
