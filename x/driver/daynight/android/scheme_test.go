package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/stretchr/testify/require"
)

func TestSchemeFromHost(t *testing.T) {
	tests := []struct {
		name string
		text string
		want daynight.Mode
		err  error
	}{
		{name: "dark", text: "dark\n", want: daynight.Dark},
		{name: "light", text: "light", want: daynight.Light},
		{name: "empty", text: "", want: daynight.Light, err: driver.ErrUnavailable},
		{name: "blank", text: " \n", want: daynight.Light, err: driver.ErrUnavailable},
		{name: "other", text: "sepia", want: daynight.Light, err: driver.ErrUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := schemeFromHost(tt.text)
			require.ErrorIs(t, err, tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestSchemeFromBit(t *testing.T) {
	dark, ok := schemeFromBit(1)
	require.True(t, ok)
	require.Equal(t, daynight.Dark, dark)
	light, ok := schemeFromBit(0)
	require.True(t, ok)
	require.Equal(t, daynight.Light, light)
	_, ok = schemeFromBit(2)
	require.False(t, ok)
}
