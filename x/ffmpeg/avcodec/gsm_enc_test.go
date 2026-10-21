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

func TestGSMEncodeSineMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	pcm, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-f", "s16le", "-",
	).Output()
	require.NoError(t, err)
	if len(pcm) != 640 {
		require.FailNow(t, fmt.Sprintf("pcm length %d", len(pcm)))
	}
	samples := make([]int16, 320)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(pcm[i*2:]))
	}

	g := newGSMCodec(Parameters{})
	var got []byte
	for f := 0; f < 2; f++ {
		frame, err := g.encodeFrame(samples[f*160 : (f+1)*160])
		require.NoError(t, err)
		got = append(got, frame[:]...)
	}

	want, err := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:sample_rate=8000:duration=0.04",
		"-c:a", "libgsm", "-f", "gsm", "-",
	).Output()
	require.NoError(t, err)
	if !bytes.Equal(got, want) {
		require.FailNow(t, fmt.Sprintf("gsm bytes mismatch\n got %x\nwant %x", got, want))
	}

	dec := newGSMCodec(Parameters{})
	for f := 0; f < 2; f++ {
		out, err := dec.decodeFrame(got[f*33 : (f+1)*33])
		require.NoError(t, err)
		if len(out) != 160 {
			require.FailNow(t, fmt.Sprintf("decoded samples %d", len(out)))
		}
	}
}
