package d3d12

import (
	"context"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestFactoryRejectsNonWindows(t *testing.T) {
	require.Equal(t, "present_d3d12", factory{}.ID())
	require.Equal(t, "Direct3D 12", factory{}.Name())
	require.Equal(t, 70, factory{}.Weight())
	if runtime.GOOS == "windows" && (runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64") {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(context.Background())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
