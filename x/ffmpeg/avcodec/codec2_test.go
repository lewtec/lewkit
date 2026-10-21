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

func TestCodec2Mode1300MatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(pcm) != 320*2 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcm)))
	}
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}

	got, err := Codec2Encode1300(samples)
	require.NoError(t, err)
	enc := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "s16le", "-ar", "8000", "-ac", "1", "-i", "pipe:0",
		"-c:a", "codec2", "-f", "codec2raw", "-",
	)
	enc.Stdin = bytes.NewReader(pcm)
	want, err := enc.Output()
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("codec2 bytes mismatch\n got %x\nwant %x", got, want))
	}
	if len(got) != 7 {
		require.FailNow(t, fmt.Sprintf("frame bytes %d", len(got)))
	}

	// codec2raw has no header; the demuxer requires the mode explicitly.
	dec := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "codec2raw", "-mode", "1300", "-i", "pipe:0",
		"-f", "s16le", "-",
	)
	dec.Stdin = bytes.NewReader(want)
	raw, err := dec.Output()
	require.NoError(t, err)
	decoded, err := Codec2Decode1300(got)
	require.NoError(t, err)
	if len(raw) != len(decoded)*2 {
		require.FailNow(t, fmt.Sprintf("decoded samples %d ffmpeg %d", len(decoded), len(raw)/2))
	}
	for i, sample := range decoded {
		wantSample := int16(binary.LittleEndian.Uint16(raw[i*2:]))
		if sample != wantSample {
			require.FailNow(t, fmt.Sprintf("sample %d got %d want %d", i, sample, wantSample))
		}
	}
}
