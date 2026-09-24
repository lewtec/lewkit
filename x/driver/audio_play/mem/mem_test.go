package mem_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/audio_play/mem"
	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestOpenRecordsPCM(t *testing.T) {
	t.Setenv("LEWKIT_AUDIO_PLAY_MEM", "1")
	t.Setenv("LEWKIT_FORCE_AUDIO_PLAY_DRIVER", "audio_play_mem")
	format := pcs.Format{Rate: 8000, Channels: 1, Sample: pcs.SampleS16LE}
	sinks, err := audio_play.Sinks(t.Context())
	require.NoError(t, err)
	require.Equal(t, []audio_play.Sink{{ID: "default", Name: "Memory"}}, sinks)

	w, err := audio_play.Open(t.Context(), audio_play.Config{Sink: "speakers", Format: format})
	require.NoError(t, err)
	buf, ok := w.(*mem.Buffer)
	require.True(t, ok)
	_, err = buf.Write([]byte{1, 0, 2, 0, 3})
	require.NoError(t, err)
	require.Equal(t, "speakers", buf.Sink())
	require.Equal(t, []byte{1, 0, 2, 0}, buf.PCM())
	require.ErrorIs(t, buf.Close(), pcs.ErrFrame)
}
