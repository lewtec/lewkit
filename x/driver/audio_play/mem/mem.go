// Package mem records playback in memory.
//
// The factory stays incompatible unless LEWKIT_AUDIO_PLAY_MEM is set, so a host
// with no audio server does not pretend to play.
package mem

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/audio_play"
	pcs "github.com/lewtec/lewkit/x/sound"
)

type factory struct{}

func (factory) ID() string   { return "audio_play_mem" }
func (factory) Name() string { return "Memory" }
func (factory) Weight() int  { return 0 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if driver.GetEnv(ctx, "LEWKIT_AUDIO_PLAY_MEM") == "" {
		return fmt.Errorf("%w: memory sound driver", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (audio_play.Driver, error) {
	return opener{}, nil
}

type opener struct{}

func (opener) Sinks(context.Context) ([]audio_play.Sink, error) {
	return []audio_play.Sink{{ID: "default", Name: "Memory"}}, nil
}

func (opener) Open(_ context.Context, cfg audio_play.Config) (io.WriteCloser, error) {
	return Open(cfg)
}

// Buffer is PCM captured by Open.
type Buffer struct {
	mu     sync.Mutex
	format pcs.Format
	sink   string
	frame  int
	buf    bytes.Buffer
	tail   []byte
	closed bool
}

// Open records writes for cfg. An empty sink is "default".
func Open(cfg audio_play.Config) (*Buffer, error) {
	frame, err := cfg.Format.Frame()
	if err != nil {
		return nil, err
	}
	sink := cfg.Sink
	if sink == "" {
		sink = "default"
	}
	return &Buffer{format: cfg.Format, sink: sink, frame: frame}, nil
}

// Sink is the device name Open stored.
func (b *Buffer) Sink() string { return b.sink }

// Format is the PCM layout Open stored.
func (b *Buffer) Format() pcs.Format { return b.format }

// PCM is the whole frames accepted so far.
func (b *Buffer) PCM() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buf.Bytes()...)
}

func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return 0, pcs.ErrClosed
	}
	buf := append(append([]byte(nil), b.tail...), p...)
	n := len(buf) - len(buf)%b.frame
	if _, err := b.buf.Write(buf[:n]); err != nil {
		return 0, err
	}
	b.tail = append(b.tail[:0], buf[n:]...)
	return len(p), nil
}

func (b *Buffer) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil
	}
	b.closed = true
	if len(b.tail) != 0 {
		return pcs.ErrFrame
	}
	return nil
}
