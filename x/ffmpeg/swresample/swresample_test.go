package swresample

import (
	"encoding/binary"
	"math"
	"testing"

	"github.com/lewtec/lewkit/x/ffmpeg/avutil"
	"github.com/stretchr/testify/require"
)

func TestS16F32AsmMatchesScalar(t *testing.T) {
	src := make([]int16, 32)
	for i := range src {
		src[i] = int16(i*1000 - 16000)
	}
	src[0] = -32768
	src[1] = 32767
	src[2] = -1
	src[3] = 1
	got := make([]float32, len(src))
	s16ToF32(got, src)
	avutil.SetScalar(true)
	t.Cleanup(func() { avutil.SetScalar(false) })
	want := make([]float32, len(src))
	s16ToF32(want, src)
	for i := range want {
		require.Equal(t, want[i], got[i], i)
	}
	back := make([]int16, len(src))
	avutil.SetScalar(false)
	f32ToS16(back, got)
	avutil.SetScalar(true)
	backScalar := make([]int16, len(src))
	f32ToS16(backScalar, got)
	require.Equal(t, backScalar, back)
	require.Equal(t, int16(32767), back[1])
	require.Equal(t, int16(-32768), back[0])
}

func TestDirectS16ToU8(t *testing.T) {
	src, err := avutil.NewAudio(avutil.SampleS16, 8000, avutil.Mono(), 4)
	require.NoError(t, err)
	putS16(src.Data[0][0:], 0)
	putS16(src.Data[0][2:], 256)
	putS16(src.Data[0][4:], -32768)
	putS16(src.Data[0][6:], 32767)
	dst, err := avutil.NewAudio(avutil.SampleU8, 8000, avutil.Mono(), 4)
	require.NoError(t, err)
	ctx, err := New(8000, avutil.SampleS16, avutil.Mono(), 8000, avutil.SampleU8, avutil.Mono())
	require.NoError(t, err)
	require.NoError(t, ctx.Convert(dst, src))
	require.Equal(t, []byte{128, 129, 0, 255}, dst.Data[0])
}

func putS16(b []byte, v int16) {
	binary.LittleEndian.PutUint16(b, uint16(v))
}

func TestFloatScale(t *testing.T) {
	v := float64(int16(16384)) * (1.0 / 32768)
	require.InDelta(t, 0.5, v, 1e-6)
	require.Equal(t, int16(16384), clipS16(int(math.RoundToEven(0.5*32768))))
}
