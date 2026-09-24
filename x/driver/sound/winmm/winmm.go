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
	"github.com/lewtec/lewkit/x/driver/sound"
	lib "github.com/lewtec/lewkit/x/ffi/native/winmm"
	pcs "github.com/lewtec/lewkit/x/sound"
)

type factory struct{}

func (factory) ID() string   { return "sound_winmm" }
func (factory) Name() string { return "WaveOut" }
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	list, err := lib.Devices()
	if err != nil {
		return nil, err
	}
	out := make([]sound.Sink, len(list))
	for i, item := range list {
		out[i] = sound.Sink{ID: item.ID, Name: item.Name}
	}
	return out, nil
}

func (backend) Open(ctx context.Context, cfg sound.Config) (io.WriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	frame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	sample, err := waveSample(cfg.Format.Sample)
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
			sink, err := sound.FindSink(id, sinks)
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
	return sound.NewFrames(frame, stream.Write, stream.Close), nil
}

func waveSample(sample pcs.Sample) (lib.Sample, error) {
	switch sample {
	case pcs.SampleS16LE:
		return lib.SampleS16LE, nil
	case pcs.SampleF32LE:
		return lib.SampleF32LE, nil
	default:
		return 0, pcs.ErrFormat
	}
}
