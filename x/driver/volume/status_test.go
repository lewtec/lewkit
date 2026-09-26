package volume

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVolumeIcon(t *testing.T) {
	cases := []struct {
		level float64
		muted bool
		want  string
	}{
		{0, false, "audio-volume-muted"},
		{0.5, true, "audio-volume-muted"},
		{0.2, false, "audio-volume-low"},
		{0.5, false, "audio-volume-medium"},
		{0.9, false, "audio-volume-high"},
	}
	for _, tc := range cases {
		require.Equal(t, tc.want, volumeIcon(tc.level, tc.muted))
	}
}
