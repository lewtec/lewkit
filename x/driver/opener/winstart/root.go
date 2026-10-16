// Package winstart launches files and URLs with cmd /c start.
package winstart

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_start" }
func (factory) Name() string { return "start" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS(build.GOOSWindows); err != nil {
		return err
	}
	return opener.RequireTool(ctx, "cmd")
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	return opener.Start(ctx, "cmd", "/c", "start", "", target)
}

func init() { driver.Register[opener.Driver](factory{}) }
