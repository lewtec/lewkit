// Package winmm plays PCM through the Windows waveOut API.
//
// Config.Sink is a decimal device index or the device name from waveOut.
// An empty sink uses WAVE_MAPPER, the system default.
package winmm

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	lib "github.com/lewtec/lewkit/x/ffi/native/winmm"
)

type factory struct{}

func (factory) ID() string   { return "audio_play_winmm" }
func (factory) Name() string { return "WaveOut" }
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
	sample, err := audio_play.MapSample(cfg.Format.Sample, lib.SampleS16LE, lib.SampleF32LE)
	if err != nil {
		return nil, err
	}
	id := cfg.Sink
	if id != "" {
		if _, err := strconv.Atoi(id); err != nil {
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
	}
	stream, err := lib.Open(ctx, id, lib.Layout{Sample: sample, Rate: cfg.Format.Rate, Channels: cfg.Format.Channels})
	if err != nil {
		return nil, err
	}
	return audio_play.NewFrames(frame, stream.Write, stream.Close), nil
}
