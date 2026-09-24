package sound

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatFrame(t *testing.T) {
	frame, err := (Format{Rate: 48000, Channels: 2, Sample: SampleS16LE}).Frame()
	require.NoError(t, err)
	require.Equal(t, 4, frame)
	_, err = (Format{Rate: 0, Channels: 1, Sample: SampleS16LE}).Frame()
	require.ErrorIs(t, err, ErrFormat)
	_, err = (Format{Rate: 48000, Channels: 1, Sample: 0}).Frame()
	require.ErrorIs(t, err, ErrFormat)
}

func TestMixSumsAndClips(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	var dst bytes.Buffer
	err := Mix(format, &dst, bytes.NewReader(pcm16(1000, 30000)), bytes.NewReader(pcm16(1000, 30000)))
	require.NoError(t, err)
	require.Equal(t, pcm16(2000, 32767), dst.Bytes())
}

func TestMixSilenceAfterEnd(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleF32LE}
	var dst bytes.Buffer
	err := Mix(format, &dst,
		bytes.NewReader(pcm32(0.5)),
		bytes.NewReader(pcm32(0.25, -0.25)),
	)
	require.NoError(t, err)
	require.Equal(t, pcm32(0.75, -0.25), dst.Bytes())
}

func TestMixerQueuesWriters(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	mixer, err := NewMixer(format)
	require.NoError(t, err)
	a, err := mixer.Add()
	require.NoError(t, err)
	b, err := mixer.Add()
	require.NoError(t, err)
	_, err = a.Write(pcm16(1, 2))
	require.NoError(t, err)
	_, err = b.Write(pcm16(3))
	require.NoError(t, err)
	buf := make([]byte, 4)
	n, err := mixer.Read(buf)
	require.NoError(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, pcm16(4, 2), buf)
	require.NoError(t, a.Close())
	require.NoError(t, b.Close())
	_, err = mixer.Read(buf)
	require.ErrorIs(t, err, io.EOF)
	_, err = a.Write(pcm16(1))
	require.ErrorIs(t, err, ErrClosed)
}

func TestMixerDropsPartialFrame(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	mixer, err := NewMixer(format)
	require.NoError(t, err)
	w, err := mixer.Add()
	require.NoError(t, err)
	_, err = w.Write([]byte{1})
	require.NoError(t, err)
	require.NoError(t, w.Close())
	buf := make([]byte, 2)
	_, err = mixer.Read(buf)
	require.ErrorIs(t, err, io.EOF)
}

func TestWAVRoundTrip(t *testing.T) {
	format := Format{Rate: 22050, Channels: 2, Sample: SampleS16LE}
	pcm := pcm16(1, -2, 3, -4)
	var buf bytes.Buffer
	require.NoError(t, WriteWAV(&buf, format, pcm))
	gotFormat, got, err := ReadWAV(&buf)
	require.NoError(t, err)
	require.Equal(t, format, gotFormat)
	require.Equal(t, pcm, got)
}

func TestWAVRejectsOddPCM(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	err := WriteWAV(&bytes.Buffer{}, format, []byte{1})
	require.ErrorIs(t, err, ErrFrame)
}

func pcm16(samples ...int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(out[i*2:], uint16(sample))
	}
	return out
}

func pcm32(samples ...float32) []byte {
	out := make([]byte, len(samples)*4)
	for i, sample := range samples {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(sample))
	}
	return out
}
