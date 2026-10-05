package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"io"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/sound"
)

//go:embed embed/cyberpunk.mp3
var cyberpunkMP3 []byte

// The clip is the loop from
// https://www.myinstants.com/en/instant/cyberpunk-sound-54201/
// (media/sounds/cyberpunk-sound.mp3).

func clip() ([]byte, sound.Format, error) {
	pipe, err := sound.Decode("cyberpunk.mp3", bytes.NewReader(cyberpunkMP3))
	if err != nil {
		return nil, sound.Format{}, err
	}
	format := pipe.Format()
	pcm, err := io.ReadAll(pipe)
	if err != nil {
		return nil, sound.Format{}, err
	}
	frame, err := format.Frame()
	if err != nil {
		return nil, sound.Format{}, err
	}
	if frame < 1 || len(pcm) < frame {
		return nil, sound.Format{}, sound.ErrFrame
	}
	pcm = pcm[:len(pcm)-len(pcm)%frame]
	return pcm, format, nil
}

type playOpener func(context.Context, audio_play.Config) (io.WriteCloser, error)

// loopClip writes pcm until ctx ends, then starts again at the first frame.
func loopClip(ctx context.Context, pcm []byte, format sound.Format, open playOpener) error {
	if open == nil {
		open = audio_play.Open
	}
	if len(pcm) == 0 {
		return sound.ErrFrame
	}
	out, err := open(ctx, audio_play.Config{Format: format})
	if err != nil {
		return err
	}
	defer out.Close()
	frame, err := format.Frame()
	if err != nil {
		return err
	}
	period := format.Rate * frame / 50
	if period < frame {
		period = frame
	}
	period -= period % frame
	at := 0
	buf := make([]byte, period)
	for {
		if err := ctx.Err(); err != nil {
			return nil
		}
		n := 0
		for n < len(buf) {
			if at >= len(pcm) {
				at = 0
			}
			c := copy(buf[n:], pcm[at:])
			if c == 0 {
				return sound.ErrFrame
			}
			n += c
			at += c
		}
		if _, err := out.Write(buf); err != nil {
			return err
		}
	}
}

func playLoop(ctx context.Context) error {
	pcm, format, err := clip()
	if err != nil {
		return err
	}
	err = loopClip(ctx, pcm, format, nil)
	if err == nil || errors.Is(err, context.Canceled) {
		return nil
	}
	if errors.Is(err, driver.ErrUnavailable) || errors.Is(err, driver.ErrIncompatible) {
		slog.Info("duck audio skipped", "err", err)
		return nil
	}
	return err
}
