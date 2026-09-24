package mp3

import (
	"bytes"
	"testing"

	"github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestDetectMP3(t *testing.T) {
	dec, ok := sound.Detect("song.mp3", nil)
	require.True(t, ok)
	require.Equal(t, "mp3", dec.Name())
	_, err := dec.Decode(bytes.NewReader([]byte("ID3not-an-mp3")))
	require.Error(t, err)
	dec, ok = sound.Detect("noext", []byte{0xff, 0xfb, 0x90, 0x00})
	require.True(t, ok)
	require.Equal(t, "mp3", dec.Name())
}
