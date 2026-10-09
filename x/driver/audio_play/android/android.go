//go:build android

package android

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
	pcs "github.com/lewtec/lewkit/x/sound"
)

var _ audio_play.Driver = backend{}

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (audio_play.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Sinks(ctx context.Context) ([]audio_play.Sink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return sinks(), nil
}

func (backend) Open(ctx context.Context, cfg audio_play.Config) (io.WriteCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	inFrame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	if _, err := audio_play.MapSample(cfg.Format.Sample, encodingPCM16, encodingPCM16); err != nil {
		return nil, err
	}
	mask, err := channelMask(cfg.Format.Channels)
	if err != nil {
		return nil, err
	}
	if cfg.Sink != "" {
		if _, err := audio_play.FindSink(cfg.Sink, sinks()); err != nil {
			return nil, err
		}
	}
	raw, err := jni.Int(jni.CallStatic("android.media.AudioTrack", "getMinBufferSize", cfg.Format.Rate, mask, encodingPCM16))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	frame := cfg.Format.Channels * 2
	buf, err := alignBuffer(raw, frame)
	if err != nil {
		return nil, err
	}
	ref, err := jni.New("android.media.AudioTrack", streamMusic, cfg.Format.Rate, mask, encodingPCM16, buf, modeStream)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	if ref == nil {
		return nil, fmt.Errorf("%w: AudioTrack", driver.ErrUnavailable)
	}
	state, err := jni.Int(ref.Call("getState"))
	if err != nil || state != stateInit {
		_, _ = ref.Call("release")
		ref.Release()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
		}
		return nil, fmt.Errorf("%w: AudioTrack state %d", driver.ErrUnavailable, state)
	}
	if _, err := ref.Call("play"); err != nil {
		_, _ = ref.Call("release")
		ref.Release()
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	tr := &track{
		ref:      ref,
		sample:   cfg.Format.Sample,
		channels: cfg.Format.Channels,
		rate:     cfg.Format.Rate,
		chunk:    buf,
		frame:    frame,
	}
	return audio_play.NewFrames(inFrame, func(p []byte) error {
		return tr.write(ctx, p)
	}, tr.end), nil
}

// track feeds AudioTrack in short non-blocking writes so the Java thread
// is not held for the whole clip.
type track struct {
	mu       sync.Mutex
	ref      *jni.Ref
	sample   pcs.Sample
	channels int
	rate     int
	chunk    int
	frame    int
	frames   int
}

func (t *track) write(ctx context.Context, p []byte) error {
	pcm, err := pcm16(t.sample, p)
	if err != nil {
		return err
	}
	off := 0
	for off < len(pcm) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n := len(pcm) - off
		t.mu.Lock()
		ref := t.ref
		if ref == nil {
			t.mu.Unlock()
			return pcs.ErrClosed
		}
		if n > t.chunk {
			n = t.chunk
		}
		wrote, err := jni.Int(ref.Call("write", pcm[off:off+n], 0, n, writeNonBlocking))
		if err != nil {
			t.mu.Unlock()
			return err
		}
		if wrote < 0 || wrote > n {
			t.mu.Unlock()
			return fmt.Errorf("%w: AudioTrack.write %d", driver.ErrUnavailable, wrote)
		}
		if t.frame > 0 {
			t.frames += wrote / t.frame
		}
		t.mu.Unlock()
		if wrote == 0 {
			timer := time.NewTimer(20 * time.Millisecond)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		off += wrote
	}
	return nil
}

func (t *track) end() error {
	t.mu.Lock()
	ref := t.ref
	t.ref = nil
	frames, rate := t.frames, t.rate
	t.mu.Unlock()
	if ref == nil {
		return nil
	}
	drain(ref, frames, rate)
	_, stopErr := ref.Call("stop")
	_, relErr := ref.Call("release")
	ref.Release()
	if stopErr != nil {
		return stopErr
	}
	return relErr
}

func drain(ref *jni.Ref, frames, rate int) {
	if ref == nil || rate <= 0 || frames <= 0 {
		return
	}
	deadline := time.Now().Add(time.Duration(frames)*time.Second/time.Duration(rate) + 500*time.Millisecond)
	for time.Now().Before(deadline) {
		head, err := jni.Int(ref.Call("getPlaybackHeadPosition"))
		if err != nil || head >= frames {
			return
		}
		wait := time.Until(deadline)
		if wait > 20*time.Millisecond {
			wait = 20 * time.Millisecond
		}
		if wait <= 0 {
			return
		}
		time.Sleep(wait)
	}
}
