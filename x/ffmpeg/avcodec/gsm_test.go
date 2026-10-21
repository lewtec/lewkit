package avcodec

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/lewtec/lewkit/x/test"
	"github.com/stretchr/testify/require"
)

func TestGSMDecodeMatchesFFmpeg(t *testing.T) {
	ff := test.Need(t, "ffmpeg")
	spec := "sine=frequency=440:sample_rate=8000:duration=0.08"
	enc := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-f", "lavfi", "-i", spec, "-c:a", "libgsm", "-f", "gsm", "-")
	gsm, err := enc.Output()
	require.NoError(t, err)
	require.NotEmpty(t, gsm)
	dec := exec.Command(ff, "-hide_banner", "-nostdin", "-loglevel", "error", "-f", "gsm", "-i", "pipe:0", "-f", "s16le", "-")
	dec.Stdin = bytes.NewReader(gsm)
	want, err := dec.Output()
	require.NoError(t, err)

	codec := newGSMCodec(Parameters{ID: IDGSM})
	frames, err := codec.Decode(&avutil.Packet{Data: gsm})
	require.NoError(t, err)
	var got []byte
	for _, frame := range frames {
		got = append(got, frame.Data[0]...)
	}
	require.Equal(t, want, got)

	avutil.SetScalar(true)
	scalar := newGSMCodec(Parameters{ID: IDGSM})
	frames, err = scalar.Decode(&avutil.Packet{Data: gsm})
	require.NoError(t, err)
	avutil.SetScalar(false)
	var slow []byte
	for _, frame := range frames {
		slow = append(slow, frame.Data[0]...)
	}
	require.Equal(t, want, slow)
}

func TestGSMMult40(t *testing.T) {
	src := make([]int16, 40)
	for i := range src {
		src[i] = int16(i*1000 - 20000)
	}
	dstA := make([]int16, 40)
	dstB := make([]int16, 40)
	avutil.SetScalar(true)
	gsmMult40(dstA, src, 11469)
	avutil.SetScalar(false)
	gsmMult40(dstB, src, 11469)
	require.Equal(t, dstA, dstB)
	var manual int16
	manual = int16(gsmMult(11469, int32(src[3])))
	require.Equal(t, manual, dstA[3])
}
