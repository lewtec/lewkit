//go:build linux

package pulse

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/sound"
	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestPlaySilence(t *testing.T) {
	format := pcs.Format{Rate: 48000, Channels: 2, Sample: pcs.SampleS16LE}
	w, err := backend{}.Open(t.Context(), sound.Config{Format: format, Name: "lewkit-test"})
	if err != nil {
		t.Skip(err)
	}
	frame, err := format.Frame()
	require.NoError(t, err)
	_, err = w.Write(make([]byte, frame*48000/50))
	require.NoError(t, err)
	require.NoError(t, w.Close())
}

func TestListSinks(t *testing.T) {
	sinks, err := backend{}.Sinks(t.Context())
	if err != nil {
		t.Skip(err)
	}
	require.NotEmpty(t, sinks)
	t.Log(sinks)
	for _, sink := range sinks {
		require.NotEmpty(t, sink.ID)
		require.NotEmpty(t, sink.Name)
	}
}
