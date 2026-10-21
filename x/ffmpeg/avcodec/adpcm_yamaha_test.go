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

func TestYamahaEncodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	const block = 256
	frame := block * 2
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	pcm = pcm[:frame*2]
	samples := make([]int16, frame)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}
	got, err := YamahaEncode(samples, block)
	require.NoError(t, err)
	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "adpcm_yamaha", "-block_size", strconv.Itoa(block), "-f", "wav", "-",
	)
	cmd.Stdin = bytes.NewReader(pcm)
	wav, err := cmd.Output()
	require.NoError(t, err)
	want := wavChunk(t, wav, "data")
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("yamaha mismatch\n got %x\nwant %x", got, want))
	}
}

func TestYamahaDecodeMatchesFFmpeg(t *testing.T) {
	data, _, want := ffmpegWAVADPCM(t, "adpcm_yamaha")
	requirePCM(t, YamahaDecode(data), want)
}
