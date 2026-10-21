package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"testing"
)

func TestTwolameSineMatchesFFmpeg(t *testing.T) {
	pcmRaw, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	const samples = 1152 * 4
	if len(pcmRaw) < samples*2 {
		t.Fatalf("pcm length %d", len(pcmRaw))
	}
	pcmRaw = pcmRaw[:samples*2]
	pcm := make([]int16, samples)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(pcmRaw[i*2:]))
	}

	got, err := TwolameEncodeMono(pcm, 44100, 192)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "libtwolame", "-b:a", "192k", "-f", "mp2", "-",
	)
	cmd.Stdin = bytes.NewReader(pcmRaw)
	want, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		n := len(got)
		if len(want) < n {
			n = len(want)
		}
		off := n
		for i := 0; i < n; i++ {
			if got[i] != want[i] {
				off = i
				break
			}
		}
		t.Fatalf("twolame bytes mismatch got %d want %d first diff %d", len(got), len(want), off)
	}
}
