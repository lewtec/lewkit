// Package host shows the message in the packaged iOS or macOS window.
package host

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_host" }
func (factory) Name() string { return "Host alert" }
func (factory) Weight() int  { return 70 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS == "android" {
		return fmt.Errorf("%w: android", driver.ErrIncompatible)
	}
	if !hostask.Available() {
		return fmt.Errorf("%w: no ask host", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	raw, err := hostask.CallStyled(ctx, hostask.KindAlert, n.Title, n.Message, n.Style)
	if err != nil {
		return err
	}
	status, _ := askwire.Split(raw)
	if status == askwire.StatusOK {
		return nil
	}
	return fmt.Errorf("%w: %s", driver.ErrUnavailable, status)
}

func init() { driver.Register[messagebox.Driver](factory{}) }

var (
	_ driver.DriverFactory[messagebox.Driver] = factory{}
	_ driver.Weighter                         = factory{}
)
