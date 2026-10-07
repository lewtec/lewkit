package android

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
)

// ErrUnavailable means this process cannot open /dev/binder.
var ErrUnavailable = errors.New("binder is unavailable")

// Client is the process-wide binder session.
type Client struct {
	call calls
}

type calls interface {
	streamVolume(ctx context.Context, stream int32) (cur, max int32, muted bool, err error)
	setStreamVolume(ctx context.Context, stream, index int32) error
	toggleStreamMute(ctx context.Context, stream int32) error
	interactive(ctx context.Context) (bool, error)
	wake(ctx context.Context) error
	sleep(ctx context.Context) error
	forceSuspend(ctx context.Context) error
	reboot(ctx context.Context) error
	shutdown(ctx context.Context) error
	lock(ctx context.Context) error
	night(ctx context.Context) (dark bool, err error)
	dataDir(ctx context.Context, packageName string) (string, error)
}

var (
	openMu sync.Mutex
	shared *Client
)

// Open returns the shared binder client.
// The first successful call opens it with ctx. A nil context is an error.
func Open(ctx context.Context) (*Client, error) {
	if ctx == nil {
		return nil, errors.New("android: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	openMu.Lock()
	defer openMu.Unlock()
	if shared != nil {
		return shared, nil
	}
	opened, err := open(ctx)
	if err != nil {
		return nil, err
	}
	shared = opened
	return shared, nil
}

// ForAndroid opens the client on an Android process.
func ForAndroid(ctx context.Context) (*Client, error) {
	if ctx == nil {
		return nil, errors.New("android: nil context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if runtime.GOOS != "android" {
		return nil, fmt.Errorf("%w: not android", ErrUnavailable)
	}
	return Open(ctx)
}

// StreamVolume reads one audio stream. cur and max are raw indexes.
func (c *Client) StreamVolume(ctx context.Context, stream int32) (cur, max int32, muted bool, err error) {
	if err = c.ready(); err != nil {
		return 0, 0, false, err
	}
	return c.call.streamVolume(ctx, stream)
}

// SetStreamVolume sets one audio stream index.
func (c *Client) SetStreamVolume(ctx context.Context, stream, index int32) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.setStreamVolume(ctx, stream, index)
}

// ToggleStreamMute flips mute on one audio stream.
func (c *Client) ToggleStreamMute(ctx context.Context, stream int32) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.toggleStreamMute(ctx, stream)
}

// Interactive reports whether the default display is awake.
func (c *Client) Interactive(ctx context.Context) (bool, error) {
	if err := c.ready(); err != nil {
		return false, err
	}
	return c.call.interactive(ctx)
}

// Wake turns the default display on.
func (c *Client) Wake(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.wake(ctx)
}

// Sleep turns the default display off.
func (c *Client) Sleep(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.sleep(ctx)
}

// ForceSuspend asks the power service to suspend immediately.
func (c *Client) ForceSuspend(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.forceSuspend(ctx)
}

// Reboot reboots the device.
func (c *Client) Reboot(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.reboot(ctx)
}

// Shutdown powers the device off.
func (c *Client) Shutdown(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.shutdown(ctx)
}

// Lock locks the keyguard.
func (c *Client) Lock(ctx context.Context) error {
	if err := c.ready(); err != nil {
		return err
	}
	return c.call.lock(ctx)
}

// Night reports whether the current configuration is night mode.
func (c *Client) Night(ctx context.Context) (bool, error) {
	if err := c.ready(); err != nil {
		return false, err
	}
	return c.call.night(ctx)
}

// DataDir returns the application data directory for packageName.
func (c *Client) DataDir(ctx context.Context, packageName string) (string, error) {
	if err := c.ready(); err != nil {
		return "", err
	}
	return c.call.dataDir(ctx, packageName)
}

func (c *Client) ready() error {
	if c == nil || c.call == nil {
		return ErrUnavailable
	}
	return nil
}
