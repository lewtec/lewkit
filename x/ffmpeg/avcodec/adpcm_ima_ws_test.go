package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIMAWSEncodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 2048 + 100
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := IMAWSEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	if err := os.WriteFile(rawIn, pcm, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "a.aud")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ws", "-f", "wsaud", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	aud, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := audPayload(t, aud)
	if !bytes.Equal(got, want) {
		t.Fatalf("ima ws mismatch got %d want %d", len(got), len(want))
	}
}

func audPayload(t *testing.T, b []byte) []byte {
	t.Helper()
	if len(b) < 12 {
		t.Fatal("aud short")
	}
	var out []byte
	off := 12
	for off+8 <= len(b) {
		n := int(binary.LittleEndian.Uint16(b[off:]))
		sig := binary.LittleEndian.Uint32(b[off+4:])
		if sig != 0x0000DEAF {
			t.Fatalf("aud signature %#x", sig)
		}
		off += 8
		if off+n > len(b) {
			t.Fatalf("aud chunk %d", n)
		}
		out = append(out, b[off:off+n]...)
		off += n
	}
	return out
}
