package sound

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeWAVSeeks(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	var raw bytes.Buffer
	require.NoError(t, WriteWAV(&raw, format, pcm16(1, 2, 3, 4)))
	pipe, err := Decode("clip.wav", bytes.NewReader(raw.Bytes()))
	require.NoError(t, err)
	_, err = pipe.Seek(2, io.SeekStart)
	require.NoError(t, err)
	got := make([]byte, 4)
	_, err = io.ReadFull(pipe, got)
	require.NoError(t, err)
	require.Equal(t, pcm16(3, 4), got)
	_, ok := Detect("nope.mp3", []byte("RIFF"))
	require.True(t, ok)
	_, ok = Detect("nope.bin", []byte("nope"))
	require.False(t, ok)
}
