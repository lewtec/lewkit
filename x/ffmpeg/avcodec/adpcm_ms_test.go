package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"os/exec"
	"strconv"
	"testing"
)

func TestMSADPCMEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const block = 256
	frame := (block-7)*2 + 2
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
	got, err := MSADPCMEncode(samples, block)
	require.NoError(t, err)
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_ms", "-block_size", strconv.Itoa(block), "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	require.NoError(t, err)
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("ms adpcm mismatch\n got %x\nwant %x", got, want))
	}
}

func TestMSADPCMDecodeMatchesFFmpeg(t *testing.T) {
	data, align, want := ffmpegWAVADPCM(t, "adpcm_ms")
	got, err := MSADPCMDecode(data, align)
	require.NoError(t, err)
	requirePCM(t, got, want)
}
