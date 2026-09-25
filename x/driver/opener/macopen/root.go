// Package macopen launches files and URLs with open.
package macopen

import (
	"context"
	"fmt"
	"runtime"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
)

type factory struct{}

func (factory) ID() string   { return "opener_open" }
func (factory) Name() string { return "open" }
func (factory) Weight() int  { return 60 }

func (factory) CheckCompatibility(context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: not darwin", driver.ErrIncompatible)
	}
	return opener.RequireTool("open")
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	return opener.Start(ctx, "open", target)
}

func init() { driver.Register[opener.Driver](factory{}) }
