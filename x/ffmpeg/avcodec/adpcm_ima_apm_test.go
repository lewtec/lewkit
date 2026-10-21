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

func TestIMAAPMEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 101
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=22050:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := IMAAPMEncode(samples)
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.apm")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_apm", "-f", "apm", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	apm, err := os.ReadFile(out)
	require.NoError(t, err)
	if len(apm) < 100 {
		require.FailNow(t, fmt.Sprintf("apm len %d", len(apm)))
	}
	if !bytes.Equal(got, apm[100:]) {
		require.FailNow(t, fmt.Sprintf("ima apm mismatch got %d want %d", len(got), len(apm)-100))
	}
	wantRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	decoded := IMAAPMDecode(apm[100:])
	if len(decoded)*2 != len(wantRaw) {
		require.FailNow(t, fmt.Sprintf("decode len %d want %d", len(decoded), len(wantRaw)/2))
	}
	for i, s := range decoded {
		want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
		if s != want {
			require.FailNow(t, fmt.Sprintf("apm decode at %d got %d want %d", i, s, want))
		}
	}
}
