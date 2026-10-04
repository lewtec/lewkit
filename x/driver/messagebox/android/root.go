// Package android shows an alert on the foreground activity.
package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/androidask"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/messagebox"
)

type factory struct{}

func (factory) ID() string   { return "messagebox_android" }
func (factory) Name() string { return "Android alert" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error { return androidask.Platform(ctx) }

func (factory) New(context.Context) (messagebox.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Show(ctx context.Context, n messagebox.Notice) error {
	raw, err := show(ctx, n.Title, n.Message)
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
