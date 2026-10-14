// Package kms opens one fullscreen window on the Linux framebuffer.
//
// Draw copies the RGBA front buffer to /dev/fb0. That node is the DRM
// fbdev device (virtio-gpu or bochs on QEMU). Weight sits below X11 and
// Wayland, so a session with DISPLAY or WAYLAND_DISPLAY keeps those.
package kms

import (
	"context"

	"github.com/lewtec/lewkit/x/driver/window"
)

type factory struct{}

func (factory) ID() string   { return "window_kms" }
func (factory) Name() string { return "KMS" }

// Weight sits below Wayland. A desktop session keeps X11 or Wayland.
// A guest with only /dev/fb0 still has this window.
func (factory) Weight() int { return 30 }

func (factory) New(context.Context) (window.Driver, error) {
	return opener{}, nil
}

type opener struct{}
