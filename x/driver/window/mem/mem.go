package mem

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_mem" }
func (factory) Name() string { return "Memory" }
func (factory) Weight() int  { return 0 }

func (factory) CheckCompatibility(context.Context) error {
	return driver.MemoryGate()
}

func (factory) New(context.Context) (window.Driver, error) {
	return opener{}, nil
}

type opener struct{}

// Open opens a memory window. Pixel tests call it so a native driver
// on this host does not win the weight contest.
func Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	return opener{}.Open(ctx, cfg)
}

func (opener) Open(_ context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	scale := window.Scale(cfg.Scale)
	buf := window.NewBuffer(scale.Device(w), scale.Device(h))
	buf.SetScale(scale)
	buf.SetFramePeriod(cfg.Period)
	return &win{Buffer: buf}, nil
}

type win struct {
	*window.Buffer
}

func (w *win) Draw() error {
	return w.Swap()
}
