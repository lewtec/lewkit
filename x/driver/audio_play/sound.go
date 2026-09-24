// Package audio_play plays interleaved PCM on one host sink.
//
//	w, err := audio_play.Open(ctx, audio_play.Config{Sink: "speakers", Format: format})
//	_, err = w.Write(pcm)
//	err = w.Close()
//
// An empty Sink is the server default. Write accepts any length and keeps a
// partial frame until the next Write. Close returns [github.com/lewtec/lewkit/x/sound.ErrFrame]
// when a partial frame is still queued. Import
// [github.com/lewtec/lewkit/x/driver/prelude] or one backend (mem, pulse, winmm, coreaudio).
package audio_play

import (
	"context"
	"io"

	"github.com/lewtec/lewkit/x/driver"
	pcs "github.com/lewtec/lewkit/x/sound"
)

// Sink is one playback device. ID is what Config.Sink accepts.
type Sink struct {
	ID   string
	Name string
}

// Config selects the device and the PCM layout of each Write.
type Config struct {
	Sink   string
	Format pcs.Format
	Name   string
}

// Driver opens host playback.
type Driver interface {
	Sinks(ctx context.Context) ([]Sink, error)
	Open(ctx context.Context, cfg Config) (io.WriteCloser, error)
}

// Sinks lists playback devices from the active driver.
func Sinks(ctx context.Context) ([]Sink, error) {
	return driver.WithResult(ctx, func(d Driver) ([]Sink, error) {
		return d.Sinks(ctx)
	})
}

// Open returns a writer that plays interleaved PCM on cfg.Sink.
func Open(ctx context.Context, cfg Config) (io.WriteCloser, error) {
	if err := cfg.Format.Validate(); err != nil {
		return nil, err
	}
	return driver.WithResult(ctx, func(d Driver) (io.WriteCloser, error) {
		return d.Open(ctx, cfg)
	})
}
