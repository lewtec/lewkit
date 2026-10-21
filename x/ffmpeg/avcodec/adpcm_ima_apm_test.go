package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIMAAPMEncodeMatchesFFmpeg(t *testing.T) {
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
	got := IMAAPMEncode(samples)
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	if err := os.WriteFile(rawIn, pcm, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "a.apm")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "22050", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_apm", "-f", "apm", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	apm, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(apm) < 100 {
		t.Fatalf("apm len %d", len(apm))
	}
	if !bytes.Equal(got, apm[100:]) {
		t.Fatalf("ima apm mismatch got %d want %d", len(got), len(apm)-100)
	}
	wantRaw, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	decoded := IMAAPMDecode(apm[100:])
	if len(decoded)*2 != len(wantRaw) {
		t.Fatalf("decode len %d want %d", len(decoded), len(wantRaw)/2)
	}
	for i, s := range decoded {
		want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
		if s != want {
			t.Fatalf("apm decode at %d got %d want %d", i, s, want)
		}
	}
}
