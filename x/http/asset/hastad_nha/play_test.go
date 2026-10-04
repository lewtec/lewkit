package hastad_nha

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/driver/audio_play/mem"
	"github.com/lewtec/lewkit/x/sound"
	_ "github.com/lewtec/lewkit/x/sound/mp3"
	"github.com/stretchr/testify/require"
)

func TestIngestAndPlay(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_MEMORY_DRIVER", "1")
	t.Setenv("LEWKIT_FORCE_AUDIO_PLAY_DRIVER", "audio_play_mem")

	pipe, err := sound.Decode(Name, bytes.NewReader(Bytes()))
	require.NoError(t, err)
	format := pipe.Format()
	require.Equal(t, 44100, format.Rate)
	require.Equal(t, 2, format.Channels)
	require.Equal(t, sound.SampleS16LE, format.Sample)
	require.Equal(t, int64(61056), pipe.Frames())
	require.Equal(t, 1384*time.Millisecond, format.Duration(pipe.Frames()).Round(time.Millisecond))

	pcm, err := io.ReadAll(pipe)
	require.NoError(t, err)
	require.Equal(t, 244224, len(pcm))
	require.Equal(t, 14288, peak(pcm))

	writer, err := audio_play.Open(t.Context(), audio_play.Config{Format: format, Name: Name})
	require.NoError(t, err)
	buf, ok := writer.(*mem.Buffer)
	require.True(t, ok)
	_, err = buf.Write(pcm)
	require.NoError(t, err)
	require.NoError(t, buf.Close())
	require.Equal(t, pcm, buf.PCM())
}

func peak(pcm []byte) int {
	max := 0
	for i := 0; i+1 < len(pcm); i += 2 {
		sample := int(int16(binary.LittleEndian.Uint16(pcm[i:])))
		if sample < 0 {
			sample = -sample
		}
		if sample > max {
			max = sample
		}
	}
	return max
}
