package wayland

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_wayland" }
func (factory) Name() string { return "Wayland" }

// Weight sits below X11. A session that still exports DISPLAY keeps that
// window. A Wayland-only session, where DISPLAY is unset, still has one.
func (factory) Weight() int { return 40 }

func (factory) New(context.Context) (window.Driver, error) {
	return opener{}, nil
}

type opener struct{}
