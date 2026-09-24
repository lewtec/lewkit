package audio_play

import (
	"testing"

	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestMapSample(t *testing.T) {
	s16, err := MapSample(pcs.SampleS16LE, 16, 32)
	require.NoError(t, err)
	require.Equal(t, 16, s16)
	f32, err := MapSample(pcs.SampleF32LE, 16, 32)
	require.NoError(t, err)
	require.Equal(t, 32, f32)
	_, err = MapSample(pcs.Sample(9), 16, 32)
	require.ErrorIs(t, err, pcs.ErrFormat)
}

func TestOpenRejectsFormat(t *testing.T) {
	_, err := Open(t.Context(), Config{})
	require.ErrorIs(t, err, pcs.ErrFormat)
}
