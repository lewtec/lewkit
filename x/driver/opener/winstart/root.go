// Package winstart launches files and URLs with cmd /c start.
package winstart

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_start" }
func (factory) Name() string { return "start" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("%w: not windows", driver.ErrIncompatible)
	}
	return opener.RequireTool("cmd")
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	return opener.Start(ctx, "cmd", "/c", "start", "", target)
}

func init() { driver.Register[opener.Driver](factory{}) }
