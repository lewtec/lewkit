//go:build linux

package portal

import (
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver/appearance"
	"github.com/stretchr/testify/require"
)

func TestSessionRead(t *testing.T) {
	src, err := open(t.Context())
	if err != nil {
		t.Skip(err)
	}
	scheme, err := src.Current(t.Context())
	require.NoError(t, err)
	require.Contains(t, []appearance.Scheme{appearance.Light, appearance.Dark}, scheme)
}

func TestSchemeOf(t *testing.T) {
	scheme, ok := schemeOf(dbus.MakeVariant(uint32(1)))
	require.True(t, ok)
	require.Equal(t, appearance.Dark, scheme)
	scheme, ok = schemeOf(dbus.MakeVariant(dbus.MakeVariant(uint32(2))))
	require.True(t, ok)
	require.Equal(t, appearance.Light, scheme)
	scheme, ok = schemeOf(uint32(0))
	require.True(t, ok)
	require.Equal(t, appearance.Light, scheme)
	_, ok = schemeOf("dark")
	require.False(t, ok)
}
