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

func TestIMAQTEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 64 * 3
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
	got := IMAQTEncode(samples)

	dir := t.TempDir()
	out := filepath.Join(dir, "want.aiff")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_ima_qt", "-f", "aiff", out,
	)
	cmd.Stdin = bytes.NewReader(pcm)
	require.NoError(t, cmd.Run())
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	want := aiffSound(t, raw)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ima qt mismatch\n got %x\nwant %x", got, want))
	}
}

func TestIMAQTDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	dir := t.TempDir()
	out := filepath.Join(dir, "a.aiff")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=0.05",
		"-c:a", "adpcm_ima_qt", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	got, err := IMAQTDecode(aiffSound(t, raw))
	require.NoError(t, err)
	want, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	requirePCM(t, got, want)
}

func aiffSound(t *testing.T, b []byte) []byte {
	t.Helper()
	i := bytes.Index(b, []byte("SSND"))
	if i < 0 || i+16 > len(b) {
		require.Fail(t, "ssnd missing")
	}
	n := int(binary.BigEndian.Uint32(b[i+4:]))
	if n < 8 || i+8+n > len(b) {
		require.FailNow(t, fmt.Sprintf("ssnd size %d", n))
	}
	return b[i+16 : i+8+n]
}
