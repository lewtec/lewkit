package avformat

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/stretchr/testify/require"
)

func TestWAVRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tone.wav")
	par := avcodec.Parameters{
		Type: avutil.MediaAudio, ID: avcodec.IDPCMS16LE,
		SampleFormat: avutil.SampleS16, SampleRate: 8000, Layout: avutil.Mono(),
		TimeBase: avutil.Q(1, 8000),
	}
	mux, err := Create(t.Context(), path, "wav", Options{})
	require.NoError(t, err)
	index, err := mux.AddStream(par)
	require.NoError(t, err)
	require.NoError(t, mux.WriteHeader())
	pcm := []byte{0, 0, 1, 0, 2, 0, 3, 0}
	require.NoError(t, mux.WritePacket(&avutil.Packet{Stream: index, Data: pcm, PTS: 0, Duration: 4, TimeBase: par.TimeBase}))
	require.NoError(t, mux.Close())

	demux, err := OpenInput(t.Context(), path, "", Options{})
	require.NoError(t, err)
	defer demux.Close()
	require.Equal(t, avcodec.IDPCMS16LE, demux.Streams()[0].Par.ID)
	pkt, err := demux.ReadPacket()
	require.NoError(t, err)
	require.Equal(t, pcm, pkt.Data)
}

func TestFrameCRCLine(t *testing.T) {
	var buf nopBuffer
	mux, err := Create(t.Context(), "-", "framecrc", Options{Stdout: &buf, Bitexact: true})
	require.NoError(t, err)
	par := avcodec.Parameters{
		Type: avutil.MediaVideo, ID: avcodec.IDRawVideo, Width: 16, Height: 16,
		Format: avutil.PixRGB24, TimeBase: avutil.Q(1, 25), SAR: avutil.Q(1, 1),
	}
	_, err = mux.AddStream(par)
	require.NoError(t, err)
	require.NoError(t, mux.WriteHeader())
	body := make([]byte, 768)
	require.NoError(t, mux.WritePacket(&avutil.Packet{Data: body, PTS: 0, DTS: 0, Duration: 1}))
	require.NoError(t, mux.Close())
	text := buf.String()
	require.NotContains(t, text, "software")
	require.Contains(t, text, "#tb 0: 1/25\n")
	require.Contains(t, text, "#dimensions 0: 16x16\n")
	require.Contains(t, text, "0,          0,          0,        1,      768, 0x")
}

type nopBuffer struct{ b []byte }

func (n *nopBuffer) Write(p []byte) (int, error) {
	n.b = append(n.b, p...)
	return len(p), nil
}

func (n *nopBuffer) String() string { return string(n.b) }

func TestGuessAndFormats(t *testing.T) {
	require.Equal(t, "wav", Guess("a.wav"))
	require.Equal(t, "mp4", Guess("a.mov"))
	require.NotEmpty(t, Formats())
	_, err := os.Stat(os.DevNull)
	require.NoError(t, err)
}
