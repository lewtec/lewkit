package experiments

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	dsound "github.com/lewtec/lewkit/x/driver/sound"
	"github.com/lewtec/lewkit/x/sound"
	"github.com/stretchr/testify/require"
)

func TestSoundUsage(t *testing.T) {
	text, err := cmd.Usage[Sound]("lewkit experiments sound")
	require.NoError(t, err)
	require.Contains(t, text, "sinks")
	require.Contains(t, text, "play")
	require.Contains(t, text, "mix")
	text, err = cmd.Usage[soundPlayCmd]("lewkit experiments sound play")
	require.NoError(t, err)
	require.Contains(t, text, "--sink")
	require.Contains(t, text, "--at")
	require.Contains(t, text, "audio file")
}

func TestSoundPlayParsesSink(t *testing.T) {
	got := cmd.ParseOK[Sound](t, "play", "--sink", "speakers", "a.wav")
	require.NotNil(t, got.Play)
	require.Equal(t, "speakers", got.Play.sink.Value())
	require.Equal(t, []string{"a.wav"}, argStrings(got.Play.files))
}

func TestFormatSinks(t *testing.T) {
	got := formatSinks([]dsound.Sink{{ID: "default", Name: "Speakers"}})
	require.Equal(t, "default\tSpeakers\n", got)
}

func TestPlayWAVMixesIntoSink(t *testing.T) {
	dir := t.TempDir()
	format := sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}
	clip := wavClip{format: format}
	left := clip.write(t, filepath.Join(dir, "a.wav"), pcm16(1000, 30000))
	right := clip.write(t, filepath.Join(dir, "b.wav"), pcm16(1000, 30000))
	var got bytes.Buffer
	err := (playback{
		sink:  "speakers",
		paths: []string{left, right},
		open: func(_ context.Context, cfg dsound.Config) (io.WriteCloser, error) {
			require.Equal(t, "speakers", cfg.Sink)
			require.Equal(t, format, cfg.Format)
			return nopCloser{&got}, nil
		},
	}).run(t.Context())
	require.NoError(t, err)
	require.Equal(t, pcm16(2000, 32767), got.Bytes())
}

func TestMixWAVWritesFile(t *testing.T) {
	dir := t.TempDir()
	format := sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}
	clip := wavClip{format: format}
	left := clip.write(t, filepath.Join(dir, "a.wav"), pcm16(1, 2))
	right := clip.write(t, filepath.Join(dir, "b.wav"), pcm16(3))
	out := filepath.Join(dir, "out.wav")
	require.NoError(t, mixWAV(t.Context(), out, 0, []string{left, right}, nil))
	file, err := os.Open(out)
	require.NoError(t, err)
	defer file.Close()
	gotFormat, pcm, err := sound.ReadWAV(file)
	require.NoError(t, err)
	require.Equal(t, format, gotFormat)
	require.Equal(t, pcm16(4, 2), pcm)
}

func TestPlayWAVSeeks(t *testing.T) {
	dir := t.TempDir()
	format := sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}
	path := (wavClip{format: format}).write(t, filepath.Join(dir, "a.wav"), pcm16(1, 2, 3))
	var got bytes.Buffer
	err := (playback{
		at:    250 * time.Microsecond,
		paths: []string{path},
		open: func(context.Context, dsound.Config) (io.WriteCloser, error) {
			return nopCloser{&got}, nil
		},
	}).run(t.Context())
	require.NoError(t, err)
	require.Equal(t, pcm16(3), got.Bytes())
}

func TestPlayStopsWhenCancelled(t *testing.T) {
	dir := t.TempDir()
	format := sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}
	path := (wavClip{format: format}).write(t, filepath.Join(dir, "a.wav"), pcm16(1, 2, 3, 4))
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var wrote int
	err := (playback{
		paths: []string{path},
		open: func(context.Context, dsound.Config) (io.WriteCloser, error) {
			return nopCloser{writerFunc(func(p []byte) (int, error) {
				wrote += len(p)
				return len(p), nil
			})}, nil
		},
	}).run(ctx)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, wrote)
}

func TestSoundTitle(t *testing.T) {
	require.Equal(t, "clip.wav", soundTitle([]string{"/tmp/clip.wav"}))
	require.Equal(t, "mix", soundTitle([]string{"a.wav", "b.ogg"}))
	require.Equal(t, "default", sinkLabel(""))
	require.Equal(t, "s16le", sampleName(sound.SampleS16LE))
}

func TestLoadClipsRejectsMismatch(t *testing.T) {
	dir := t.TempDir()
	a := (wavClip{format: sound.Format{Rate: 8000, Channels: 1, Sample: sound.SampleS16LE}}).write(t, filepath.Join(dir, "a.wav"), pcm16(1))
	b := (wavClip{format: sound.Format{Rate: 16000, Channels: 1, Sample: sound.SampleS16LE}}).write(t, filepath.Join(dir, "b.wav"), pcm16(1))
	_, err := loadClips([]string{a, b})
	require.ErrorIs(t, err, errAudioFormat)
	_, err = loadClips(nil)
	require.ErrorIs(t, err, errAudioFile)
}

type wavClip struct {
	format sound.Format
}

func (c wavClip) write(t *testing.T, path string, pcm []byte) string {
	t.Helper()
	file, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, sound.WriteWAV(file, c.format, pcm))
	require.NoError(t, file.Close())
	return path
}

type nopCloser struct{ io.Writer }

func (nopCloser) Close() error { return nil }

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func pcm16(samples ...int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, sample := range samples {
		out[i*2] = byte(sample)
		out[i*2+1] = byte(sample >> 8)
	}
	return out
}
