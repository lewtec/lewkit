package gtkui

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestRequireNeedsDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	err := Require(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	if runtime.GOOS == "linux" {
		require.ErrorContains(t, err, "DISPLAY")
	}
}

func TestRunCallsHere(t *testing.T) {
	err := Run(t.Context(), func() error { return errors.New("here") })
	require.EqualError(t, err, "here")
}
