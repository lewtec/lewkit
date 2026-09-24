// Package coreaudio plays PCM through AudioQueue on macOS.
//
// Config.Sink is a CoreAudio device UID or the device's display name.
// An empty sink uses the default output device.
package coreaudio

import (
	"context"
	"fmt"
	"io"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	lib "github.com/lewtec/lewkit/x/ffi/native/coreaudio"
	pcs "github.com/lewtec/lewkit/x/sound"
)

type factory struct{}

func (factory) ID() string   { return "audio_play_coreaudio" }
func (factory) Name() string { return "CoreAudio" }
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	list, err := lib.Devices()
	if err != nil {
		return nil, err
	}
	out := make([]audio_play.Sink, len(list))
	for i, item := range list {
		out[i] = audio_play.Sink{ID: item.ID, Name: item.Name}
	}
	return out, nil
}

func (backend) Open(ctx context.Context, cfg audio_play.Config) (io.WriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	frame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	sample, err := queueSample(cfg.Format.Sample)
	if err != nil {
		return nil, err
	}
	id := cfg.Sink
	if id != "" {
		sinks, err := backend{}.Sinks(ctx)
		if err != nil {
			return nil, err
		}
		sink, err := audio_play.FindSink(id, sinks)
		if err != nil {
			return nil, err
		}
		id = sink.ID
	}
	stream, err := lib.Open(ctx, id, lib.Layout{Sample: sample, Rate: cfg.Format.Rate, Channels: cfg.Format.Channels})
	if err != nil {
		return nil, err
	}
	return audio_play.NewFrames(frame, stream.Write, stream.Close), nil
}

func queueSample(sample pcs.Sample) (lib.Sample, error) {
	switch sample {
	case pcs.SampleS16LE:
		return lib.SampleS16LE, nil
	case pcs.SampleF32LE:
		return lib.SampleF32LE, nil
	default:
		return 0, pcs.ErrFormat
	}
}
