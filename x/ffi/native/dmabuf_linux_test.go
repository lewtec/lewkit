//go:build linux && !android

package native

import "testing"

func TestUseHardwareAccelDegradesWhenTheSmokeFails(t *testing.T) {
	if useHardwareAccel(accelSmoke{}) {
		t.Fatal("a display that did not initialize kept the GPU")
	}
	if useHardwareAccel(accelSmoke{display: true, importExt: true}) {
		t.Fatal("a failed DMA-BUF import kept the GPU")
	}
	if !useHardwareAccel(accelSmoke{display: true}) {
		t.Fatal("a display without DMA-BUF import was forced to software")
	}
	if !useHardwareAccel(accelSmoke{display: true, importExt: true, image: true}) {
		t.Fatal("a successful import was forced to software")
	}
}

func TestDMABufSmokeDoesNotAbort(t *testing.T) {
	_ = DMABufSmoke()
}
