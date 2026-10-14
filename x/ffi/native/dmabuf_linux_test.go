//go:build linux && !android

package native

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUseHardwareAccelDegradesWhenTheSmokeFails(t *testing.T) {
	if useHardwareAccel(accelSmoke{}) {
		require.FailNow(t, fmt.Sprint("a display that did not initialize kept the GPU"))
	}
	if useHardwareAccel(accelSmoke{display: true, importExt: true}) {
		require.FailNow(t, fmt.Sprint("a failed DMA-BUF import kept the GPU"))
	}
	if !useHardwareAccel(accelSmoke{display: true}) {
		require.FailNow(t, fmt.Sprint("a display without DMA-BUF import was forced to software"))
	}
	if !useHardwareAccel(accelSmoke{display: true, importExt: true, image: true}) {
		require.FailNow(t, fmt.Sprint("a successful import was forced to software"))
	}
}

func TestDMABufSmokeDoesNotAbort(t *testing.T) {
	_ = DMABufSmoke()
}
