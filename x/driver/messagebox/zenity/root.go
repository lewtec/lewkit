// Package zenity shows an error dialog with the zenity command.
package zenity

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_zenity" }
func (factory) Name() string { return "Zenity alert" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if driver.GetEnv(ctx, "DISPLAY") == "" && driver.GetEnv(ctx, "WAYLAND_DISPLAY") == "" {
		return fmt.Errorf("%w: neither DISPLAY nor WAYLAND_DISPLAY set", driver.ErrIncompatible)
	}
	return execdriver.RequireBinary(ctx, "zenity")
}

func (factory) New(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	return execdriver.Run(ctx, execdriver.MustCommand("zenity", zenityFlag(n.Style), "--title", n.Title, "--text", n.Message))
}

func zenityFlag(style string) string {
	switch messagebox.NormalizeStyle(style) {
	case messagebox.StyleWarning:
		return "--warning"
	case messagebox.StyleCritical:
		return "--error"
	default:
		return "--info"
	}
}

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
