package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"testing"
)

func TestG722MatchesFFmpeg(t *testing.T) {
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=16000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 640*2 {
		t.Fatalf("pcm length %d", len(pcm))
	}
	samples := make([]int16, 640)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}

	got, err := G722Encode(samples)
	if err != nil {
		t.Fatal(err)
	}
	enc := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "16000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "g722", "-f", "g722", "-",
	)
	enc.Stdin = bytes.NewReader(pcm)
	want, err := enc.Output()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("g722 bytes mismatch\n got %x\nwant %x", got, want)
	}

	dec := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "g722", "-i", "pipe:0", "-f", "s16le", "-",
	)
	dec.Stdin = bytes.NewReader(want)
	raw, err := dec.Output()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := G722Decode(got)
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
