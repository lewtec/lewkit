package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestSWFEncodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 4096 * 2
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) < samplesN*2 {
		t.Fatalf("pcm length %d", len(pcm))
	}
	pcm = pcm[:samplesN*2]
	samples := make([]int16, samplesN)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got := SWFEncode(samples)
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_swf", "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		t.Fatalf("swf mismatch got %d want %d\n got %x\nwant %x", len(got), len(want), got[:32], want[:32])
	}
}

func TestSWFDecodeMatchesFFmpeg(t *testing.T) {
	const samplesN = 4096 * 2
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
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
	wavPath := filepath.Join(dir, "a.wav")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", rawIn,
		"-c:a", "adpcm_swf", wavPath,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	wav, err := os.ReadFile(wavPath)
	if err != nil {
		t.Fatal(err)
	}
	got, err := SWFDecode(wavChunk(t, wav, "data"))
	if err != nil {
		t.Fatal(err)
	}
	want, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", wavPath, "-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	requirePCM(t, got, want)
}
