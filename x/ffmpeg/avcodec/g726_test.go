package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"testing"
)

func TestG726MatchesFFmpeg(t *testing.T) {
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 320*2 {
		t.Fatalf("pcm length %d", len(pcm))
	}
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := G726Encode(samples, 4)
	if err != nil {
		t.Fatal(err)
	}
	enc := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "g726", "-f", "g726", "-",
	)
	enc.Stdin = bytes.NewReader(pcm)
	want, err := enc.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("g726 bytes mismatch\n got %x\nwant %x", got, want)
	}
	dec := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "g726", "-i", "pipe:0", "-f", "s16le", "-",
	)
	dec.Stdin = bytes.NewReader(want)
	raw, err := dec.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := G726Decode(got, 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != len(decoded)*2 {
		t.Fatalf("decoded samples %d ffmpeg %d", len(decoded), len(raw)/2)
	}
	for i, sample := range decoded {
		wantSample := int16(binary.LittleEndian.Uint16(raw[i*2:]))
		if sample != wantSample {
			t.Fatalf("sample %d got %d want %d", i, sample, wantSample)
		}
	}
}
