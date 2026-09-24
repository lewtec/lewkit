// Package pulse plays PCM through libpulse-simple.
//
// Config.Sink is the PulseAudio sink name. An empty name is the server default.
// PipeWire serves the same API through pipewire-pulse.
package pulse

import (
	"context"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	lib "github.com/lewtec/lewkit/x/ffi/native/pulse"
)

type factory struct{}

func (factory) ID() string   { return "audio_play_pulse" }
func (factory) Name() string { return "PulseAudio" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	if err := lib.Available(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (audio_play.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Sinks(ctx context.Context) ([]audio_play.Sink, error) {
	list, err := lib.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]audio_play.Sink, len(list))
	for i, item := range list {
		name := item.Description
		if name == "" {
			name = item.Name
		}
		out[i] = audio_play.Sink{ID: item.Name, Name: name}
	}
	return out, nil
}

func (backend) Open(ctx context.Context, cfg audio_play.Config) (io.WriteCloser, error) {
	frame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	sample, err := audio_play.MapSample(cfg.Format.Sample, lib.SampleS16LE, lib.SampleF32LE)
	if err != nil {
		return nil, err
	}
	stream, err := lib.Playback(ctx, cfg.Name, cfg.Sink, cfg.Name, sample, cfg.Format.Rate, cfg.Format.Channels)
	if err != nil {
		return nil, err
	}
	return audio_play.NewFrames(frame, stream.Write, stream.Close), nil
}
