package avcodec

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIMAWAVEncodeMatchesFFmpeg(t *testing.T) {
	const block = 256
	frame := (block-4)*2 + 1
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
	got, err := IMAWAVEncode(samples, block)
	if err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_ima_wav", "-block_size", strconv.Itoa(block), "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		t.Fatalf("ima wav mismatch\n got %x\nwant %x", got, want)
	}
}

func TestIMAWAVDecodeMatchesFFmpeg(t *testing.T) {
	data, align, want := ffmpegWAVADPCM(t, "adpcm_ima_wav")
	got, err := IMAWAVDecode(data, align)
	if err != nil {
		t.Fatal(err)
	}
	requirePCM(t, got, want)
}

func ffmpegWAVADPCM(t *testing.T, codec string) (data []byte, align int, pcm []byte) {
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "a.wav")
	cmd := exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=0.1",
		"-c:a", codec, out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatal(err, string(b))
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	data = wavChunk(t, raw, "data")
	fmtc := wavChunk(t, raw, "fmt ")
	if len(fmtc) < 14 {
		t.Fatal("fmt short")
	}
	align = int(binary.LittleEndian.Uint16(fmtc[12:14]))
	pcm, err = exec.Command(
		"ffmpeg", "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	if err != nil {
		t.Fatal(err)
	}
	return data, align, pcm
}

func s16Bytes(samples []int16) []byte {
	b := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.LittleEndian.PutUint16(b[i*2:], uint16(s))
	}
	return b
}

func requirePCM(t *testing.T, got []int16, want []byte) {
	t.Helper()
	gb := s16Bytes(got)
	if bytes.Equal(gb, want) {
		return
	}
	n := len(gb)
	if len(want) < n {
		n = len(want)
	}
	for i := 0; i+1 < n; i += 2 {
		if gb[i] != want[i] || gb[i+1] != want[i+1] {
			t.Fatalf("pcm mismatch at sample %d got %d want %d (lens %d %d)", i/2,
				int16(binary.LittleEndian.Uint16(gb[i:])),
				int16(binary.LittleEndian.Uint16(want[i:])),
				len(gb)/2, len(want)/2)
		}
	}
	t.Fatalf("pcm length got %d want %d", len(gb)/2, len(want)/2)
}
