package d3d12

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnavailableOffWindows(t *testing.T) {
	if runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		t.Skip()
	}
	require.ErrorIs(t, Available(), ErrUnavailable)
	_, err := OpenNative(2, 1, 8, 8)
	require.ErrorIs(t, err, ErrUnavailable)
	_, err = OpenDevice()
	require.ErrorIs(t, err, ErrUnavailable)
}
