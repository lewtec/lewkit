package avcodec

import (
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestFLACRoundTripAndFFmpeg(t *testing.T) {
	bin := test.Need(t, "ffmpeg")
	const n = 4096
	frame, err := avutil.NewAudio(avutil.SampleS16, 44100, avutil.Mono(), n)
	require.NoError(t, err)
	for i := 0; i < n; i++ {
		v := int16((i*17)%1000 - 500)
		binary.LittleEndian.PutUint16(frame.Data[0][i*2:], uint16(v))
	}
	enc, err := OpenEncoder(Parameters{ID: IDFLAC, Type: avutil.MediaAudio, SampleRate: 44100, Layout: avutil.Mono(), SampleFormat: avutil.SampleS16})
	require.NoError(t, err)
	pkts, err := enc.Encode(frame)
	require.NoError(t, err)
	rest, err := enc.Flush()
	require.NoError(t, err)
	pkts = append(pkts, rest...)
	require.Len(t, pkts, 1)

	dec, err := OpenDecoder(Parameters{ID: IDFLAC, Type: avutil.MediaAudio})
	require.NoError(t, err)
	got, err := dec.Decode(pkts[0])
	require.NoError(t, err)
	require.Equal(t, frame.Data[0], got[0].Data[0][:n*2])

	dir := t.TempDir()
	path := filepath.Join(dir, "a.flac")
	require.NoError(t, os.WriteFile(path, pkts[0].Data, 0o644))
	out := filepath.Join(dir, "a.pcm")
	cmd := exec.CommandContext(t.Context(), bin, "-hide_banner", "-y", "-i", path, "-f", "s16le", "-ac", "1", out)
	log, err := cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	pcm, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, frame.Data[0], pcm)

	ref := filepath.Join(dir, "ref.flac")
	raw := filepath.Join(dir, "in.pcm")
	require.NoError(t, os.WriteFile(raw, frame.Data[0], 0o644))
	cmd = exec.CommandContext(t.Context(), bin, "-hide_banner", "-y", "-f", "s16le", "-ar", "44100", "-ac", "1", "-i", raw, "-c:a", "flac", ref)
	log, err = cmd.CombinedOutput()
	require.NoError(t, err, string(log))
	refBytes, err := os.ReadFile(ref)
	require.NoError(t, err)
	got, err = dec.Decode(&avutil.Packet{Data: refBytes})
	require.NoError(t, err)
	require.Equal(t, frame.Data[0], got[0].Data[0][:n*2])
}
