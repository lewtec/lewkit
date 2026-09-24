package ogg

import (
	"io"
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestDecodeOgg(t *testing.T) {
	file, err := os.Open("testdata/test.ogg")
	require.NoError(t, err)
	defer file.Close()
	pipe, err := sound.Decode("testdata/test.ogg", file)
	require.NoError(t, err)
	require.Equal(t, sound.SampleF32LE, pipe.Format().Sample)
	require.Positive(t, pipe.Frames())
	_, err = pipe.Seek(pipe.Frames()/2, io.SeekStart)
	require.NoError(t, err)
	buf := make([]byte, pipe.Format().Channels*4)
	_, err = io.ReadFull(pipe, buf)
	require.NoError(t, err)
}
