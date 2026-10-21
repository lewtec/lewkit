package avcodec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
	"os/exec"
	"testing"
)

func TestShineSineMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	pcmRaw, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=44100:duration=1",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	const samples = 1152 * 4
	if len(pcmRaw) < samples*2 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcmRaw)))
	}
	pcmRaw = pcmRaw[:samples*2]
	pcm := make([]int16, samples)
	for i := range pcm {
		pcm[i] = int16(binary.LittleEndian.Uint16(pcmRaw[i*2:]))
	}

	got, err := ShineEncodeMono(pcm, 44100, 128)
	require.NoError(t, err)

	cmd := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "44100", "-ac", "1", "-i", "pipe:0",
		"-c:a", "libshine", "-b:a", "128k", "-f", "mp3", "-",
	)
	cmd.Stdin = bytes.NewReader(pcmRaw)
	want, err := cmd.Output()
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("shine bytes mismatch got %d want %d", len(got), len(want)))
	}
}
