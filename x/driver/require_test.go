package driver_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

func TestRequireEnv(t *testing.T) {
	ctx := driver.WithEnv(t.Context(), []string{"FOO=1"})
	require.NoError(t, driver.RequireEnv(ctx, "FOO"))
	err := driver.RequireEnv(ctx, "MISSING")
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestRequireEnvFallsBackToProcess(t *testing.T) {
	t.Setenv("LEWKIT_REQUIRE_TEST", "yes")
	require.NoError(t, driver.RequireEnv(t.Context(), "LEWKIT_REQUIRE_TEST"))
}

func TestRequireAnyEnv(t *testing.T) {
	ctx := driver.WithEnv(t.Context(), []string{"WAYLAND_DISPLAY=wayland-0"})
	require.NoError(t, driver.RequireAnyEnv(ctx, "DISPLAY", "WAYLAND_DISPLAY"))
	err := driver.RequireAnyEnv(ctx, "DISPLAY", "OTHER")
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestRequireTermux(t *testing.T) {
	t.Setenv("TERMUX_VERSION", "")
	require.ErrorIs(t, driver.RequireTermux(), driver.ErrIncompatible)
	require.False(t, driver.IsTermux())

	t.Setenv("TERMUX_VERSION", "0.118")
	require.NoError(t, driver.RequireTermux())
	require.True(t, driver.IsTermux())
}

func TestGetEnvContextOverridesProcess(t *testing.T) {
	t.Setenv("FOO", "process")
	ctx := driver.WithEnv(t.Context(), []string{"FOO=ctx"})
	require.Equal(t, "ctx", driver.GetEnv(ctx, "FOO"))
	require.Equal(t, "process", driver.GetEnv(t.Context(), "FOO"))
}
