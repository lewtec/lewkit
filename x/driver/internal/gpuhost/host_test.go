package gpuhost

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestMetalHost(t *testing.T) {
	err := Metal()
	if runtime.GOOS == "darwin" || runtime.GOOS == "ios" {
		return
	}
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.ErrorContains(t, err, "not apple")
}

func TestDirect3DHost(t *testing.T) {
	err := Direct3D()
	if runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		return
	}
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.ErrorContains(t, err, "not windows")
}
