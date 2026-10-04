// Package ios opens a URL or file through the packaged iOS host.
package ios

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/iosbox"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_ios" }
func (factory) Name() string { return "iOS open" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "ios" {
		return fmt.Errorf("%w: not ios", driver.ErrIncompatible)
	}
	if !iosbox.Available() {
		return fmt.Errorf("%w: no ios host", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, err := iosbox.Call(ctx, iosbox.Request{Op: iosbox.OpOpen, Target: target})
	if err != nil {
		return err
	}
	status, _, err := iosbox.Result(raw)
	if err != nil {
		return err
	}
	if status != askwire.StatusOK {
		return fmt.Errorf("%w: open", driver.ErrUnavailable)
	}
	return nil
}

func init() { driver.Register[opener.Driver](factory{}) }

var (
	_ driver.DriverFactory[opener.Driver] = factory{}
	_ driver.Weighter                     = factory{}
)
