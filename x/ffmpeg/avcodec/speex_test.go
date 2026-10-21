package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"testing"
)

func TestSpeexNBMatchesFFmpeg(t *testing.T) {
	pcmRaw, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcmRaw) < 640 {
		t.Fatalf("pcm length %d", len(pcmRaw))
	}
	pcmRaw = pcmRaw[:640]
	pcm := make([]int16, 320)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(pcmRaw[i*2:]))
	}

	got, err := SpeexEncodeNB(pcm)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "libspeex", "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcmRaw)
	wav, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := speexWAVData(wav)
	if want == nil {
		t.Fatalf("wav data chunk missing, len %d", len(wav))
	}
	if !bytes.Equal(got, want) {
		n := len(got)
		if len(want) < n {
			n = len(want)
		}
		off := 0
		for off < n && got[off] == want[off] {
			off++
		}
		t.Fatalf("speex bytes mismatch got %d want %d first differ %d\n got %x\nwant %x", len(got), len(want), off, got, want)
	}
}

func speexWAVData(b []byte) []byte {
	if len(b) < 12 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return nil
	}
	i := 12
	for i+8 <= len(b) {
		id := string(b[i : i+4])
		sz := binary.LittleEndian.Uint32(b[i+4:])
		i += 8
		if id == "data" {
			if sz == 0xFFFFFFFF {
				return b[i:]
			}
			end := i + int(sz)
			if end > len(b) {
				return b[i:]
			}
			return b[i:end]
		}
		i += int(sz)
		if sz&1 != 0 {
			i++
		}
	}
	return nil
}
