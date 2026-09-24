package mem_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/sound"
	"github.com/lewtec/lewkit/x/driver/sound/mem"
	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestOpenRecordsPCM(t *testing.T) {
	t.Setenv("LEWKIT_SOUND_MEM", "1")
	t.Setenv("LEWKIT_FORCE_SOUND_DRIVER", "sound_mem")
	format := pcs.Format{Rate: 8000, Channels: 1, Sample: pcs.SampleS16LE}
	sinks, err := sound.Sinks(t.Context())
	require.NoError(t, err)
	require.Equal(t, []sound.Sink{{ID: "default", Name: "Memory"}}, sinks)

	w, err := sound.Open(t.Context(), sound.Config{Sink: "speakers", Format: format})
	require.NoError(t, err)
	buf, ok := w.(*mem.Buffer)
	require.True(t, ok)
	_, err = buf.Write([]byte{1, 0, 2, 0, 3})
	require.NoError(t, err)
	require.Equal(t, "speakers", buf.Sink())
	require.Equal(t, []byte{1, 0, 2, 0}, buf.PCM())
	require.ErrorIs(t, buf.Close(), pcs.ErrFrame)
}
