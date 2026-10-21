package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIMAALPEncodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 2048 + 101
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=22050:duration=1",
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
	got := IMAALPEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	if err := os.WriteFile(rawIn, pcm, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "a.pcm")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_alp", "-f", "alp", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	alp, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(alp) < 20 || string(alp[:4]) != "ALP " {
		t.Fatalf("alp header %q len %d", alp[:min(4, len(alp))], len(alp))
	}
	want := alp[20:]
	if !bytes.Equal(got, want) {
		t.Fatalf("ima alp mismatch got %d want %d", len(got), len(want))
	}
}
