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

func TestIMAALPDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 101
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=22050:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	for _, ext := range []string{".pcm", ".tun"} {
		out := filepath.Join(dir, "a"+ext)
		cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
			"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
			"-c:a", "adpcm_ima_alp", "-f", "alp", out,
		)
		if b, err := cmd.CombinedOutput(); err != nil {
			require.NoError(t, err, string(b))
		}
		alp, err := os.ReadFile(out)
		require.NoError(t, err)
		hdr := 16
		if ext == ".pcm" {
			hdr = 20
		}
		got := IMAALPDecode(alp[hdr:])
		wantRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
			"-i", out, "-f", "s16le", "-",
		).Output()
		require.NoError(t, err)
		if len(got)*2 != len(wantRaw) {
			require.FailNow(t, fmt.Sprintf("%s ima alp decode len %d want %d", ext, len(got), len(wantRaw)/2))
		}
		for i := range got {
			want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
			if got[i] != want {
				require.FailNow(t, fmt.Sprintf("%s ima alp decode at %d got %d want %d", ext, i, got[i], want))
			}
		}
	}
}

func TestIMAALPEncodeMatchesFFmpeg(t *testing.T) {
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
	got := IMAALPEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.pcm")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_alp", "-f", "alp", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	alp, err := os.ReadFile(out)
	require.NoError(t, err)
	if len(alp) < 20 || string(alp[:4]) != "ALP " {
		require.FailNow(t, fmt.Sprintf("alp header %q len %d", alp[:min(4, len(alp))], len(alp)))
	}
	want := alp[20:]
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ima alp mismatch got %d want %d", len(got), len(want)))
	}
}
