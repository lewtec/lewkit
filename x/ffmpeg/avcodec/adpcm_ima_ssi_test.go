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

func TestIMASSIDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 100
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.vag")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ssi", "-f", "kvag", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	vag, err := os.ReadFile(out)
	require.NoError(t, err)
	if len(vag) < 14 {
		require.Fail(t, "kvag short")
	}
	got := IMASSIDecode(vag[14:])
	wantRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(got)*2 != len(wantRaw) {
		require.FailNow(t, fmt.Sprintf("ima ssi decode len %d want %d", len(got), len(wantRaw)/2))
	}
	for i := range got {
		want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
		if got[i] != want {
			require.FailNow(t, fmt.Sprintf("ima ssi decode at %d got %d want %d", i, got[i], want))
		}
	}
}

func TestIMASSIEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 100
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := IMASSIEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.vag")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ssi", "-f", "kvag", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	vag, err := os.ReadFile(out)
	require.NoError(t, err)
	if len(vag) < 14 || string(vag[:4]) != "KVAG" {
		require.FailNow(t, fmt.Sprintf("kvag header %q", vag[:min(14, len(vag))]))
	}
	want := vag[14:]
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ima ssi mismatch got %d want %d", len(got), len(want)))
	}
}
