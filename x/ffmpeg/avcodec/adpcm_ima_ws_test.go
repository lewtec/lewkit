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

func TestIMAWSEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 100
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := IMAWSEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.aud")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ws", "-f", "wsaud", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	aud, err := os.ReadFile(out)
	require.NoError(t, err)
	want := audPayload(t, aud)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ima ws mismatch got %d want %d", len(got), len(want)))
	}
}

func TestIMAWSDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const samplesN = 2048 + 100
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:samplesN*2]
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	require.NoError(t, os.WriteFile(rawIn, pcm, 0o644))
	out := filepath.Join(dir, "a.aud")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ws", "-f", "wsaud", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	aud, err := os.ReadFile(out)
	require.NoError(t, err)
	got := IMAWSDecode(audPayload(t, aud))
	wantRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	want := make([]int16, len(wantRaw)/2)
	for i := range want {
		want[i] = int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
	}
	if len(got) != len(want) {
		require.FailNow(t, fmt.Sprintf("ima ws decode len %d want %d", len(got), len(want)))
	}
	for i := range want {
		if got[i] != want[i] {
			require.FailNow(t, fmt.Sprintf("ima ws decode at %d got %d want %d", i, got[i], want[i]))
		}
	}
}

func audPayload(t *testing.T, b []byte) []byte {
	t.Helper()
	if len(b) < 12 {
		require.Fail(t, "aud short")
	}
	var out []byte
	off := 12
	for off+8 <= len(b) {
		n := int(binary.LittleEndian.Uint16(b[off:]))
		sig := binary.LittleEndian.Uint32(b[off+4:])
		if sig != 0x0000DEAF {
			require.FailNow(t, fmt.Sprintf("aud signature %#x", sig))
		}
		off += 8
		if off+n > len(b) {
			require.FailNow(t, fmt.Sprintf("aud chunk %d", n))
		}
		out = append(out, b[off:off+n]...)
		off += n
	}
	return out
}
