package experiments

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lewtec/lewkit/x/cmd"
	dsound "github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/sound"
	_ "github.com/lewtec/lewkit/x/sound/prelude"
	"github.com/lewtec/lewkit/x/taskgroup"
)

var (
	errAudioFile   = errors.New("audio file required")
	errAudioFormat = errors.New("audio format mismatch")
)

// Sound is `lewkit experiments sound`.
type Sound struct {
	Sinks *soundSinksCmd
	Play  *soundPlayCmd
	Mix   *soundMixCmd
}

func (Sound) Description() string {
	return "list sinks, play audio files, and mix them"
}

type soundSinksCmd struct{}

func (soundSinksCmd) Description() string { return "list playback devices" }

func (*soundSinksCmd) Run(ctx context.Context) error {
	list, err := dsound.Sinks(ctx)
	if err != nil {
		return err
	}
	slog.Info("sinks", "count", len(list))
	for _, sink := range list {
		slog.Info("sink", "id", sink.ID, "name", sink.Name)
	}
	_, err = os.Stdout.WriteString(formatSinks(list))
	return err
}

func formatSinks(sinks []dsound.Sink) string {
	var b strings.Builder
	for _, sink := range sinks {
		fmt.Fprintf(&b, "%s\t%s\n", sink.ID, sink.Name)
	}
	return b.String()
}

type soundPlayCmd struct {
	sink  cmd.StringArg   `long:"sink" default:"" help:"playback device; empty uses the server default"`
	at    cmd.DurationArg `long:"at" default:"0s" help:"start position"`
	files []cmd.StringArg `help:"audio file; - reads stdin"`
}

func (soundPlayCmd) Description() string {
	return "play one file, or mix several, into a sink"
}

func (c *soundPlayCmd) Run(ctx context.Context) error {
	p := playback{sink: c.sink.Value(), at: c.at.Value(), paths: argStrings(c.files), open: dsound.Open}
	return runDemo(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, soundTitle(p.paths), taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
			p.status = st
			return p.run(ctx)
		})
		return nil
	})
}

type playOpener func(context.Context, dsound.Config) (io.WriteCloser, error)

type playback struct {
	sink   string
	at     time.Duration
	paths  []string
	open   playOpener
	status *taskgroup.Status
}

func (p playback) run(ctx context.Context) error {
	clips, err := loadClips(p.paths)
	if err != nil {
		return err
	}
	defer clips.close()
	pipe, err := openPipeline(clips.pipes, p.at)
	if err != nil {
		return err
	}
	slog.Info("decoded",
		"files", p.paths,
		"rate", clips.format.Rate,
		"channels", clips.format.Channels,
		"sample", sampleName(clips.format.Sample),
		"duration", pipe.Duration().Round(time.Millisecond),
	)
	if p.status != nil {
		p.status.Update("opening " + sinkLabel(p.sink))
		p.status.Progress(0, pipe.Frames())
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	out, err := p.open(ctx, dsound.Config{Sink: p.sink, Format: clips.format})
	if err != nil {
		return err
	}
	slog.Info("play", "sink", sinkLabel(p.sink), "frames", pipe.Frames())
	writeErr := copyPeriods(ctx, out, pipe, p.status)
	if writeErr == nil {
		slog.Info("done", "sink", sinkLabel(p.sink))
	}
	err = out.Close()
	if writeErr != nil {
		err = writeErr
	}
	return err
}

// copyPeriods reads and writes one 20ms period at a time.
// The playback device blocks while its period is still sounding, so this loop
// stays on the device clock. A cancelled ctx returns before the next period.
func copyPeriods(ctx context.Context, dst io.Writer, src *sound.Pipeline, status *taskgroup.Status) error {
	format := src.Format()
	frame, err := format.Frame()
	if err != nil {
		return err
	}
	period := format.Rate * frame / 50
	if period < frame {
		period = frame
	}
	period -= period % frame
	total := src.Frames()
	var played int64
	buf := make([]byte, period)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, readErr := src.Read(buf)
		if n > 0 {
			if _, err := dst.Write(buf[:n]); err != nil {
				return err
			}
			played += int64(n / frame)
			reportPlay(status, format, played, total)
		}
		if readErr == io.EOF {
			reportPlay(status, format, played, total)
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

func reportPlay(status *taskgroup.Status, format sound.Format, played, total int64) {
	if status == nil {
		return
	}
	status.Progress(played, total)
	status.Update(format.Duration(played).Round(time.Second).String())
}

func sinkLabel(sink string) string {
	if sink == "" {
		return "default"
	}
	return sink
}

func sampleName(sample sound.Sample) string {
	switch sample {
	case sound.SampleS16LE:
		return "s16le"
	case sound.SampleF32LE:
		return "f32le"
	default:
		return "pcm"
	}
}

func soundTitle(paths []string) string {
	switch len(paths) {
	case 0:
		return "sound"
	case 1:
		if paths[0] == "-" {
			return "stdin"
		}
		return filepath.Base(paths[0])
	default:
		return "mix"
	}
}

type soundMixCmd struct {
	out   cmd.StringArg   `long:"out" help:"WAV file to write"`
	at    cmd.DurationArg `long:"at" default:"0s" help:"start position"`
	files []cmd.StringArg `help:"audio file; - reads stdin"`
}

func (soundMixCmd) Description() string { return "mix audio files into one WAV file" }

func (c *soundMixCmd) Run(ctx context.Context) error {
	paths := argStrings(c.files)
	return runDemo(ctx, func(ctx context.Context) error {
		taskgroup.Go(ctx, soundTitle(paths), taskgroup.IO, func(ctx context.Context, st *taskgroup.Status) error {
			return mixWAV(ctx, c.out.Value(), c.at.Value(), paths, st)
		})
		return nil
	})
}

func mixWAV(ctx context.Context, outPath string, at time.Duration, paths []string, status *taskgroup.Status) error {
	clips, err := loadClips(paths)
	if err != nil {
		return err
	}
	defer clips.close()
	pipe, err := openPipeline(clips.pipes, at)
	if err != nil {
		return err
	}
	slog.Info("mix",
		"files", paths,
		"out", outPath,
		"rate", clips.format.Rate,
		"channels", clips.format.Channels,
		"sample", sampleName(clips.format.Sample),
		"duration", pipe.Duration().Round(time.Millisecond),
	)
	if status != nil {
		status.Update("mixing")
		status.Progress(0, pipe.Frames())
	}
	var mixed bytes.Buffer
	if err := copyPeriods(ctx, &mixed, pipe, status); err != nil {
		return err
	}
	file, err := os.Create(outPath)
	if err != nil {
		return err
	}
	err = sound.WriteWAV(file, clips.format, mixed.Bytes())
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func argStrings(args []cmd.StringArg) []string {
	out := make([]string, len(args))
	for i, arg := range args {
		out[i] = arg.Value()
	}
	return out
}

type clips struct {
	format sound.Format
	pipes  []*sound.Pipeline
	close  func()
}

func loadClips(paths []string) (clips, error) {
	if len(paths) == 0 {
		return clips{}, errAudioFile
	}
	var out clips
	var closers []func() error
	out.close = func() {
		for _, close := range closers {
			close()
		}
	}
	for i, path := range paths {
		pipe, close, err := decodePath(path)
		if close != nil {
			closers = append(closers, close)
		}
		if err != nil {
			out.close()
			return clips{}, err
		}
		if i == 0 {
			out.format = pipe.Format()
		} else if pipe.Format() != out.format {
			out.close()
			return clips{}, fmt.Errorf("%w: %s", errAudioFormat, path)
		}
		out.pipes = append(out.pipes, pipe)
	}
	return out, nil
}

func decodePath(path string) (*sound.Pipeline, func() error, error) {
	if path == "-" {
		pipe, err := sound.Decode("-", os.Stdin)
		return pipe, nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	pipe, err := sound.Decode(path, file)
	if err != nil {
		file.Close()
		return nil, nil, err
	}
	return pipe, file.Close, nil
}

func openPipeline(pipes []*sound.Pipeline, at time.Duration) (*sound.Pipeline, error) {
	var pipe *sound.Pipeline
	var err error
	if len(pipes) == 1 {
		pipe = pipes[0]
	} else {
		pipe, err = sound.Merge(pipes...)
		if err != nil {
			return nil, err
		}
	}
	frames, err := pipe.Format().FramesFor(at)
	if err != nil {
		return nil, err
	}
	if frames == 0 {
		return pipe, nil
	}
	if _, err := pipe.Seek(frames, io.SeekStart); err != nil {
		return nil, err
	}
	return pipe, nil
}
