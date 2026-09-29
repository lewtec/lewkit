package android

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/daynight"
	"github.com/stretchr/testify/require"
)

func TestModeFromUI(t *testing.T) {
	const mask, yes = 0x30, 0x20
	tests := []struct {
		name   string
		uiMode int
		want   daynight.Mode
	}{
		{name: "undefined", uiMode: 0, want: daynight.Light},
		{name: "night no", uiMode: 0x10, want: daynight.Light},
		{name: "night yes", uiMode: 0x20, want: daynight.Dark},
		{name: "night yes with type", uiMode: 0x21, want: daynight.Dark},
		{name: "zero mask", uiMode: 0x20, want: daynight.Light},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMask, gotYes := mask, yes
			if tt.name == "zero mask" {
				gotMask = 0
			}
			require.Equal(t, tt.want, modeFromUI(tt.uiMode, gotMask, gotYes))
		})
	}
}
