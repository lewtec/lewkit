package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSWFEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 4096 * 2
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(pcm) < samplesN*2 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcm)))
	}
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := SWFEncode(samples)
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_swf", "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	require.NoError(t, err)
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("swf mismatch got %d want %d\n got %x\nwant %x", len(got), len(want), got[:32], want[:32]))
	}
}

func TestSWFDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 4096 * 2
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	wavPath := filepath.Join(dir, "a.wav")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_swf", wavPath,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	wav, err := os.ReadFile(wavPath)
	require.NoError(t, err)
	got, err := SWFDecode(wavChunk(t, wav, "data"))
	require.NoError(t, err)
	want, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", wavPath, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	requirePCM(t, got, want)
}
