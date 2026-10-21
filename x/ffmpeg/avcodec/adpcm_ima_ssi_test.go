package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestIMASSIDecodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 2048 + 100
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	pcm = pcm[:samplesN*2]
	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	if err := os.WriteFile(rawIn, pcm, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "a.vag")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ssi", "-f", "kvag", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	vag, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(vag) < 14 {
		t.Fatal("kvag short")
	}
	got := IMASSIDecode(vag[14:])
	wantRaw, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(got)*2 != len(wantRaw) {
		t.Fatalf("ima ssi decode len %d want %d", len(got), len(wantRaw)/2)
	}
	for i := range got {
		want := int16(binary.LittleEndian.Uint16(wantRaw[i*2:]))
		if got[i] != want {
			t.Fatalf("ima ssi decode at %d got %d want %d", i, got[i], want)
		}
	}
}

func TestIMASSIEncodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 2048 + 100
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1",
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
	got := IMASSIEncode(samples)

	dir := t.TempDir()
	rawIn := filepath.Join(dir, "a.s16")
	if err := os.WriteFile(rawIn, pcm, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "a.vag")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_ima_ssi", "-f", "kvag", out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	vag, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(vag) < 14 || string(vag[:4]) != "KVAG" {
		t.Fatalf("kvag header %q", vag[:min(14, len(vag))])
	}
	want := vag[14:]
	if !bytes.Equal(got, want) {
		t.Fatalf("ima ssi mismatch got %d want %d", len(got), len(want))
	}
}
