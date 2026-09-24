package sound

import (
	"bytes"
	"testing"

	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestFramesKeepsTail(t *testing.T) {
	var got bytes.Buffer
	w := NewFrames(2, func(p []byte) error {
		_, err := got.Write(p)
		return err
	}, func() error { return nil })
	_, err := w.Write([]byte{1, 0, 2, 0, 3})
	require.NoError(t, err)
	require.Equal(t, []byte{1, 0, 2, 0}, got.Bytes())
	require.ErrorIs(t, w.Close(), pcs.ErrFrame)
	_, err = w.Write([]byte{1, 0})
	require.ErrorIs(t, err, pcs.ErrClosed)
}

func TestFindSinkPrefersID(t *testing.T) {
	sinks := []Sink{{ID: "1", Name: "Speakers"}, {ID: "2", Name: "HDMI"}}
	got, err := FindSink("1", sinks)
	require.NoError(t, err)
	require.Equal(t, "Speakers", got.Name)
	got, err = FindSink("HDMI", sinks)
	require.NoError(t, err)
	require.Equal(t, "2", got.ID)
	_, err = FindSink("missing", sinks)
	require.ErrorIs(t, err, ErrSink)
}
