package gpuhost

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestMetalHost(t *testing.T) {
	test.IncompatibleOff(t, func(context.Context) error { return Metal() }, build.GOOSDarwin, build.GOOSIOS)
	require.ErrorContains(t, Metal(), "not apple")
}

func TestDirect3DHost(t *testing.T) {
	host := build.HostGOOS()
	arch := build.HostGOARCH()
	if host == build.GOOSWindows && (arch == build.GOARCHAMD64 || arch == build.GOARCHARM64) {
		return
	}
	err := Direct3D()
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.ErrorContains(t, err, "not windows")
}
