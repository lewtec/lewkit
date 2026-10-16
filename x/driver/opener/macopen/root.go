// Package macopen launches files and URLs with open.
package macopen

import (
	"context"

	"github.com/lewtec/lewkit/x/build"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_open" }
func (factory) Name() string { return "open" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.ForGOOS(build.GOOSDarwin); err != nil {
		return err
	}
	return opener.RequireTool(ctx, "open")
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	return opener.Start(ctx, "open", target)
}

func init() { driver.Register[opener.Driver](factory{}) }
