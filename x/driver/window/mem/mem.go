package mem

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_mem" }
func (factory) Name() string { return "Memory" }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (window.Driver, error) {
	return opener{}, nil
}

type opener struct{}

func (opener) Open(_ context.Context, cfg window.Config) (window.Window, error) {
	w, h, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	return &win{Buffer: window.NewBuffer(w, h)}, nil
}

type win struct {
	*window.Buffer
}

func (w *win) Draw() error {
	if w.Closed() {
		return window.ErrClosed
	}
	return nil
}
