package avutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReduceAndRescale(t *testing.T) {
	got, exact := Reduce(2, 4, math.MaxInt32)
	require.True(t, exact)
	require.Equal(t, Rational{Num: 1, Den: 2}, got)
	require.Equal(t, int64(40000), Rescale(1, 1000000, 25))
	require.Equal(t, Rational{Num: 1, Den: 3}, Q(1, 2).Mul(Q(2, 3)))
	require.Equal(t, int32(0), Q(1, 2).Cmp(Q(2, 4)))
	require.Equal(t, int32(-1), Q(1, 3).Cmp(Q(1, 2)))
}

func TestParseSizeRateTime(t *testing.T) {
	width, height, err := ParseSize("hd720")
	require.NoError(t, err)
	require.Equal(t, 1280, width)
	require.Equal(t, 720, height)
	rate, err := ParseRate("30000/1001")
	require.NoError(t, err)
	require.Equal(t, Q(30000, 1001), rate)
	us, err := ParseTimeUS("00:00:01.5")
	require.NoError(t, err)
	require.Equal(t, int64(1500000), us)
	color, err := ParseColor("red")
	require.NoError(t, err)
	require.Equal(t, [4]byte{255, 0, 0, 255}, color)
}

func TestExpr(t *testing.T) {
	expr, err := ParseExpr("if(gt(iw,ih),iw/2,ih)", "iw", "ih")
	require.NoError(t, err)
	require.InDelta(t, 160, expr.Eval(map[string]float64{"iw": 320, "ih": 240}), 0.001)
	expr, err = ParseExpr("clip(n,0,10)+PI")
	require.NoError(t, err)
	require.InDelta(t, 10+math.Pi, expr.Eval(map[string]float64{"n": 40}), 0.001)
}

func TestBswap(t *testing.T) {
	buf := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06}
	Bswap16(buf)
	require.Equal(t, []byte{0x02, 0x01, 0x04, 0x03, 0x06, 0x05}, buf)
	buf = []byte{0x01, 0x02, 0x03, 0x04}
	Bswap32(buf)
	require.Equal(t, []byte{0x04, 0x03, 0x02, 0x01}, buf)
}

func TestLayout(t *testing.T) {
	layout, err := ParseLayout("5.1")
	require.NoError(t, err)
	require.Equal(t, 6, layout.Channels())
	layout, err = ParseLayout("stereo")
	require.NoError(t, err)
	require.Equal(t, Stereo(), layout)
}
