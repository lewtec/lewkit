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
	"github.com/lewtec/lewkit/x/driver/sound"
	lib "github.com/lewtec/lewkit/x/ffi/native/pulse"
	pcs "github.com/lewtec/lewkit/x/sound"
)

type factory struct{}

func (factory) ID() string   { return "sound_pulse" }
func (factory) Name() string { return "PulseAudio" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	if err := lib.Available(); err != nil {
		return fmt.Errorf("%w: %v", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (sound.Driver, error) {
	return backend{}, nil
}

type backend struct{}

func (backend) Sinks(ctx context.Context) ([]sound.Sink, error) {
	list, err := lib.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]sound.Sink, len(list))
	for i, item := range list {
		name := item.Description
		if name == "" {
			name = item.Name
		}
		out[i] = sound.Sink{ID: item.Name, Name: name}
	}
	return out, nil
}

func (backend) Open(ctx context.Context, cfg sound.Config) (io.WriteCloser, error) {
	frame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	sample, err := pulseSample(cfg.Format.Sample)
	if err != nil {
		return nil, err
	}
	stream, err := lib.Playback(ctx, cfg.Name, cfg.Sink, cfg.Name, sample, cfg.Format.Rate, cfg.Format.Channels)
	if err != nil {
		return nil, err
	}
	return sound.NewFrames(frame, stream.Write, stream.Close), nil
}

func pulseSample(sample pcs.Sample) (lib.Sample, error) {
	switch sample {
	case pcs.SampleS16LE:
		return lib.SampleS16LE, nil
	case pcs.SampleF32LE:
		return lib.SampleF32LE, nil
	default:
		return 0, pcs.ErrFormat
	}
}
