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

func TestChooseSwapchainPrerotatesLandscape(t *testing.T) {
	// The phone is sideways. currentExtent is the landscape size the user
	// sees. The swapchain image is the identity size, and the picture is
	// stored with the same 90° turn the compositor inverts.
	caps := surfaceCaps{
		currentW: 2400, currentH: 1080,
		currentTrans: transform90,
		transforms:   transformIdent | transform90 | transform180 | transform270,
	}
	w, h, pre, content := chooseSwapchain(1080, 2400, 0, 0, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transform90), content)
	gotW, gotH := presentExtent(2400, 1080, content)
	assert.Equal(t, w, gotW)
	assert.Equal(t, h, gotH)

	caps.currentTrans = transform270
	w, h, pre, content = chooseSwapchain(2400, 1080, 1080, 2400, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform270), pre)
	assert.Equal(t, uint32(transform270), content)
}

func TestChooseSwapchainLeavesPortraitUpright(t *testing.T) {
	w, h, pre, content := chooseSwapchain(1080, 2400, 0, 0, surfaceCaps{
		currentW: 1080, currentH: 2400, currentTrans: transformIdent, transforms: transformIdent,
	})
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transformIdent), pre)
	assert.Equal(t, uint32(transformIdent), content)
}

func TestChooseSwapchainTurnsUpsideDown(t *testing.T) {
	caps := surfaceCaps{
		currentW: 1080, currentH: 2400,
		currentTrans: transform180,
		transforms:   transformIdent | transform180,
	}
	w, h, pre, content := chooseSwapchain(1080, 2400, 0, 0, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform180), pre)
	assert.Equal(t, uint32(transform180), content)
}

func TestChooseSwapchainPrerotatesAnUndefinedExtent(t *testing.T) {
	caps := surfaceCaps{
		currentW: extentUnset, currentH: extentUnset,
		minW: 1, minH: 1, maxW: 4096, maxH: 4096,
		currentTrans: transform90,
		transforms:   transformIdent | transform90,
	}
	w, h, pre, content := chooseSwapchain(800, 400, 0, 0, caps)
	assert.Equal(t, 400, w)
	assert.Equal(t, 800, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transform90), content)
}

func TestHoldMisalignedWaitsForTheExtentAspect(t *testing.T) {
	assert.True(t, holdMisaligned(2400, 1080, 1080, 2400, true))
	assert.False(t, holdMisaligned(2400, 1080, 2400, 1080, true))
	assert.False(t, holdMisaligned(2400, 1080, 1080, 2400, false))
	assert.False(t, holdMisaligned(1080, 2400, extentUnset, extentUnset, true))
}

func TestChooseSwapchainMatchesAFreshTurn(t *testing.T) {
	// The view is already landscape. The current extent is still the
	// portrait size. Drawing that extent stretches it into the new window.
	// This frame follows the request, and the turned image is the window.
	caps := surfaceCaps{
		currentW: 1080, currentH: 2400,
		currentTrans: transform90,
		transforms:   transformIdent | transform90 | transform180 | transform270,
	}
	w, h, pre, content := chooseSwapchain(2400, 1080, 1080, 2400, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transform90), content)
	gotW, gotH := presentExtent(2400, 1080, content)
	assert.Equal(t, w, gotW)
	assert.Equal(t, h, gotH)
}

func TestChooseSwapchainDoesNotKeepTheIdentityRequest(t *testing.T) {
	// The request is still the portrait buffer. The extent is the landscape
	// window. Following that request would stay sideways.
	caps := surfaceCaps{
		currentW: 2400, currentH: 1080,
		currentTrans: transform90,
		transforms:   transformIdent | transform90 | transform180 | transform270,
	}
	w, h, pre, content := chooseSwapchain(1080, 2400, 1080, 2400, caps)
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
	assert.Equal(t, uint32(transform90), pre)
	assert.Equal(t, uint32(transform90), content)
	gotW, gotH := presentExtent(2400, 1080, content)
	assert.Equal(t, w, gotW)
	assert.Equal(t, h, gotH)
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

func TestOrientedUndoesATurnedImage(t *testing.T) {
	screen := &Screen{outW: 1080, outH: 2400, turn: transform90}
	w, h := screen.Oriented()
	assert.Equal(t, 2400, w)
	assert.Equal(t, 1080, h)
	screen.turn = transformIdent
	w, h = screen.Oriented()
	assert.Equal(t, 1080, w)
	assert.Equal(t, 2400, h)
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
