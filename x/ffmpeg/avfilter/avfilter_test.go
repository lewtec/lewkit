package avfilter

import (
	"os"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avcodec"
	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestTestSrcMatchesFFmpeg(t *testing.T) {
	bin := test.Need(t, "ffmpeg")
	graph, err := Parse("testsrc=size=320x240:rate=25:duration=1", 0)
	require.NoError(t, err)
	frame, err := graph.Next()
	require.NoError(t, err)
	enc, err := avcodec.OpenEncoder(avcodec.Parameters{ID: avcodec.IDRawVideo, Type: avutil.MediaVideo, Width: 320, Height: 240, Format: avutil.PixRGB24})
	require.NoError(t, err)
	pkts, err := enc.Encode(frame)
	require.NoError(t, err)
	dir := t.TempDir()
	out := dir + "/ref.rgb"
	cmd := exec.CommandContext(t.Context(), bin, "-nostdin", "-hide_banner", "-loglevel", "error", "-cpuflags", "0",
		"-f", "lavfi", "-i", "testsrc=size=320x240:rate=25:duration=1",
		"-frames:v", "1", "-pix_fmt", "rgb24", "-f", "rawvideo", out)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	ref, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, ref, pkts[0].Data)
}

func TestSineMatchesFFmpeg(t *testing.T) {
	bin := test.Need(t, "ffmpeg")
	graph, err := Parse("sine=frequency=440:sample_rate=44100:duration=1", 0)
	require.NoError(t, err)
	frame, err := graph.Next()
	require.NoError(t, err)
	require.Equal(t, 1024, frame.NbSamples)
	dir := t.TempDir()
	out := dir + "/ref.pcm"
	cmd := exec.CommandContext(t.Context(), bin, "-nostdin", "-hide_banner", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-frames:a", "1", "-f", "s16le", out)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	ref, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, ref, frame.Data[0][:2048])
}
