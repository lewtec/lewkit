package sound

import (
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPipelineSeek(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	pipe, err := New(format, pcm16(1, 2, 3, 4))
	require.NoError(t, err)
	require.Equal(t, int64(4), pipe.Frames())
	require.Equal(t, 500*time.Microsecond, pipe.Duration())
	at, err := pipe.Seek(-1, io.SeekEnd)
	require.NoError(t, err)
	require.Equal(t, int64(3), at)
	got := make([]byte, 2)
	_, err = pipe.Read(got)
	require.NoError(t, err)
	require.Equal(t, pcm16(4), got)
	_, err = pipe.Read(got)
	require.ErrorIs(t, err, io.EOF)
	_, err = pipe.Seek(5, io.SeekStart)
	require.ErrorIs(t, err, ErrSeek)
}

func TestPipelineGainAndTake(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	pipe, err := New(format, pcm16(1000, 2000, 3000, 4000))
	require.NoError(t, err)
	pipe.Gain(0.5)
	require.NoError(t, pipe.Take(1, 2))
	got, err := io.ReadAll(pipe)
	require.NoError(t, err)
	require.Equal(t, pcm16(1000, 1500), got)
	at, err := pipe.Seek(0, io.SeekStart)
	require.NoError(t, err)
	require.Equal(t, int64(0), at)
	_, err = io.ReadFull(pipe, got[:2])
	require.NoError(t, err)
	require.Equal(t, pcm16(1000), got[:2])
}

func TestMergeSeek(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	left, err := New(format, pcm16(1, 2))
	require.NoError(t, err)
	right, err := New(format, pcm16(3))
	require.NoError(t, err)
	merged, err := Merge(left, right)
	require.NoError(t, err)
	_, err = merged.Seek(1, io.SeekStart)
	require.NoError(t, err)
	got, err := io.ReadAll(merged)
	require.NoError(t, err)
	require.Equal(t, pcm16(2), got)
}

func TestFramesFor(t *testing.T) {
	format := Format{Rate: 8000, Channels: 1, Sample: SampleS16LE}
	frames, err := format.FramesFor(1500 * time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, int64(12000), frames)
	_, err = format.FramesFor(-time.Millisecond)
	require.ErrorIs(t, err, ErrSeek)
}
