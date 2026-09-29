package android

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNightFromUIMode(t *testing.T) {
	dark, ok := NightFromUIMode(uiModeNightYes | 0x01)
	require.True(t, ok)
	require.True(t, dark)

	dark, ok = NightFromUIMode(uiModeNightNo)
	require.True(t, ok)
	require.False(t, dark)

	_, ok = NightFromUIMode(0)
	require.False(t, ok)
}

func TestNightFromSetting(t *testing.T) {
	dark, ok := NightFromSetting(nightModeYes)
	require.True(t, ok)
	require.True(t, dark)

	dark, ok = NightFromSetting(nightModeNo)
	require.True(t, ok)
	require.False(t, dark)

	_, ok = NightFromSetting(0)
	require.False(t, ok)
}
