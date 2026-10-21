package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"testing"
)

func TestGSMMSEncodeMatchesFFmpeg(t *testing.T) {
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.08",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) != 1280 {
		t.Fatalf("pcm length %d", len(pcm))
	}
	samples := make([]int16, 640)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	g := newGSMCodec(Parameters{})
	var got []byte
	for f := 0; f < 2; f++ {
		frame, err := g.encodeMS(samples[f*320 : (f+1)*320])
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, frame[:]...)
	}

	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "libgsm_ms", "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		t.Fatalf("gsm_ms bytes mismatch\n got %x\nwant %x", got, want)
	}
}

func wavChunk(t *testing.T, b []byte, id string) []byte {
	t.Helper()
	if len(b) < 12 || string(b[:4]) != "RIFF" {
		t.Fatalf("wav header %q", b[:min(12, len(b))])
	}
	off := 12
	for off+8 <= len(b) {
		name := string(b[off : off+4])
		n := int(binary.LittleEndian.Uint32(b[off+4:]))
		off += 8
		if n == int(^uint32(0)) {
			n = len(b) - off
		}
		if off+n > len(b) {
			t.Fatalf("wav chunk %s overruns", name)
		}
		if name == id {
			return b[off : off+n]
		}
		off += n
		if n%2 == 1 {
			off++
		}
	}
	t.Fatalf("wav chunk %s missing", id)
	return nil
}
