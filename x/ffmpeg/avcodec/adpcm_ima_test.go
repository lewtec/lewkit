package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
)

func TestIMAWAVEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const block = 256
	frame := (block-4)*2 + 1
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(pcm) < frame*2 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcm)))
	}
	pcm = pcm[:frame*2]
	samples := make([]int16, frame)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := IMAWAVEncode(samples, block)
	require.NoError(t, err)

	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_ima_wav", "-block_size", strconv.Itoa(block), "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	require.NoError(t, err)
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ima wav mismatch\n got %x\nwant %x", got, want))
	}
}

func TestIMAWAVDecodeMatchesFFmpeg(t *testing.T) {
	data, align, want := ffmpegWAVADPCM(t, "adpcm_ima_wav")
	got, err := IMAWAVDecode(data, align)
	require.NoError(t, err)
	requirePCM(t, got, want)
}

func ffmpegWAVADPCM(t *testing.T, codec string) (data []byte, align int, pcm []byte) {
	ff := test.Need(t, "ffmpeg")
	t.Helper()
	dir := t.TempDir()
	out := filepath.Join(dir, "a.wav")
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=0.1",
		"-c:a", codec, out,
	)
	if b, err := cmd.CombinedOutput(); err != nil {
		require.NoError(t, err, string(b))
	}
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	data = wavChunk(t, raw, "data")
	fmtc := wavChunk(t, raw, "fmt ")
	if len(fmtc) < 14 {
		require.Fail(t, "fmt short")
	}
	align = int(binary.LittleEndian.Uint16(fmtc[12:14]))
	pcm, err = exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-i", out, "-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
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
			require.FailNow(t, fmt.Sprintf("pcm mismatch at sample %d got %d want %d (lens %d %d)", i/2,
				int16(binary.LittleEndian.Uint16(gb[i:])),
				int16(binary.LittleEndian.Uint16(want[i:])),
				len(gb)/2, len(want)/2))
		}
	}
	require.FailNow(t, fmt.Sprintf("pcm length got %d want %d", len(gb)/2, len(want)/2))
}
