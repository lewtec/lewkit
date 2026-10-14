package x11

import (
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestMonitorOn(t *testing.T) {
	if !monitorOn("Monitor is On") || monitorOn("Monitor is Off") {
		require.FailNow(t, fmt.Sprint("monitor"))
	}
}

func TestMissingDisplay(t *testing.T) {
	ctx := driver.WithEnv(t.Context(), []string{"DISPLAY="})
	err := factory{}.CheckCompatibility(ctx)
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
