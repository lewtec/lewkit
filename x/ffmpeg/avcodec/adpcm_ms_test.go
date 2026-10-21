package avcodec

import (
	"bytes"
	"encoding/binary"
	"os/exec"
	"strconv"
	"testing"
)

func TestMSADPCMEncodeMatchesFFmpeg(t *testing.T) {
	const block = 256
	frame := (block-7)*2 + 2
	pcm, err := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	if len(pcm) < frame*2 {
		t.Fatalf("pcm length %d", len(pcm))
	}
	pcm = pcm[:frame*2]
	samples := make([]int16, frame)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := MSADPCMEncode(samples, block)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_ms", "-block_size", strconv.Itoa(block), "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		t.Fatalf("ms adpcm mismatch\n got %x\nwant %x", got, want)
	}
}

func TestMSADPCMDecodeMatchesFFmpeg(t *testing.T) {
	data, align, want := ffmpegWAVADPCM(t, "adpcm_ms")
	got, err := MSADPCMDecode(data, align)
	if err != nil {
		t.Fatal(err)
	}
	requirePCM(t, got, want)
}
