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

func TestARGOEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 32*2 + 10
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
	got := ARGOEncode(samples)
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.asf")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_argo", "-f", "argo_asf", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	asf, err := os.ReadFile(out)
	require.NoError(t, err)
	if len(asf) < 44 {
		require.FailNow(t, fmt.Sprintf("asf len %d", len(asf)))
	}
	if !bytes.Equal(got, asf[44:]) {
		require.FailNow(t, fmt.Sprintf("argo mismatch got %d want %d", len(got), len(asf)-44))
	}
	wantRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "argo_asf", "-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	decoded := ARGODecode(asf[44:])
	if len(decoded)*2 != len(wantRaw) {
		require.FailNow(t, fmt.Sprintf("decode len %d want %d", len(decoded), len(wantRaw)/2))
	}
	for i, s := range decoded {
		want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
		if s != want {
			require.FailNow(t, fmt.Sprintf("argo decode at %d got %d want %d", i, s, want))
		}
	}
}
