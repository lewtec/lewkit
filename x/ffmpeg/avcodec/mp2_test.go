package avcodec

import (
	"bytes"
	"encoding/binary"
	"math"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestMP2FixedMatchesFFmpeg(t *testing.T) {
	bin := test.Need(t, "ffmpeg")
	const samples = 1152 + 512
	frame, err := avutil.NewAudio(avutil.SampleS16, 44100, avutil.Mono(), samples)
	require.NoError(t, err)
	for i := 0; i < samples; i++ {
		value := int16(math.Sin(2*math.Pi*440*float64(i)/44100) * 32767)
		binary.LittleEndian.PutUint16(frame.Data[0][i*2:], uint16(value))
	}
	codec := newMP2(Parameters{
		Type: avutil.MediaAudio, SampleRate: 44100,
		Layout: avutil.Mono(), SampleFormat: avutil.SampleS16,
	})
	packets, err := codec.Encode(frame)
	require.NoError(t, err)
	rest, err := codec.Flush()
	require.NoError(t, err)
	packets = append(packets, rest...)
	var got []byte
	for _, packet := range packets {
		got = append(got, packet.Data...)
	}

	cmd := exec.CommandContext(t.Context(), bin,
		"-hide_banner", "-nostdin", "-loglevel", "error", "-y",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "mp2fixed", "-f", "mp2", "-")
	cmd.Stdin = bytes.NewReader(frame.Data[0])
	reference, err := cmd.Output()
	if exit, ok := err.(*exec.ExitError); ok {
		require.NoError(t, err, string(exit.Stderr))
	}
	require.NoError(t, err)
	require.Equal(t, -1, mp2FirstDiff(reference, got), "got %d bytes, ffmpeg %d", len(got), len(reference))
	require.Equal(t, reference, got)
}

func mp2FirstDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	if len(a) != len(b) {
		return n
	}
	return -1
}
