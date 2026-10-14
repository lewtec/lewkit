package sway

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestDPMSOn(t *testing.T) {
	if !dpmsOn(`{"dpms": true}`) || dpmsOn(`{"dpms": false}`) {
		require.FailNow(t, fmt.Sprint("dpms"))
	}
}

func TestMissingWayland(t *testing.T) {
	ctx := driver.WithEnv(t.Context(), []string{"WAYLAND_DISPLAY="})
	err := factory{}.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
