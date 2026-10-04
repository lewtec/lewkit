package android

import (
	"encoding/binary"
	"math"
	"testing"

	pcs "github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestChannelMask(t *testing.T) {
	mask, err := channelMask(1)
	require.NoError(t, err)
	require.Equal(t, channelMono, mask)

	mask, err = channelMask(2)
	require.NoError(t, err)
	require.Equal(t, channelStereo, mask)

	_, err = channelMask(3)
	require.ErrorIs(t, err, pcs.ErrFormat)
}

func TestAlignBuffer(t *testing.T) {
	n, err := alignBuffer(8, 4)
	require.NoError(t, err)
	require.Equal(t, 8, n)

	n, err = alignBuffer(5, 4)
	require.NoError(t, err)
	require.Equal(t, 8, n)

	_, err = alignBuffer(0, 4)
	require.Error(t, err)
	_, err = alignBuffer(-2, 4)
	require.Error(t, err)
}

func TestPCM16(t *testing.T) {
	s16 := []byte{1, 2, 3, 4}
	got, err := pcm16(pcs.SampleS16LE, s16)
	require.NoError(t, err)
	require.Equal(t, s16, got)

	var in [8]byte
	binary.LittleEndian.PutUint32(in[0:], math.Float32bits(1))
	binary.LittleEndian.PutUint32(in[4:], math.Float32bits(-1))
	got, err = pcm16(pcs.SampleF32LE, in[:])
	require.NoError(t, err)
	require.Equal(t, []byte{0xff, 0x7f, 0x01, 0x80}, got)

	_, err = pcm16(0, s16)
	require.ErrorIs(t, err, pcs.ErrFormat)

	_, err = pcm16(pcs.SampleF32LE, []byte{1, 2, 3})
	require.ErrorIs(t, err, pcs.ErrFrame)
}

func TestSinks(t *testing.T) {
	list := sinks()
	require.Equal(t, sinkID, list[0].ID)
	require.Equal(t, "Default", list[0].Name)
}
