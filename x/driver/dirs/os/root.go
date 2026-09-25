// Package host resolves app directories from the OS homes.
package host

import (
	"context"
	"fmt"
	"os"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/dirs"
)

type factory struct{}

func (factory) ID() string   { return "dirs_host" }
func (factory) Name() string { return "OS directories" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (dirs.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Resolve(_ context.Context, appID string) (dirs.Dirs, error) {
	got, err := livePaths(appID)
	if err != nil {
		return dirs.Dirs{}, err
	}
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs.Dirs{}, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return dirs.Dirs{Data: got.Data, Cache: got.Cache, Config: got.Config, Inbox: got.Inbox}, nil
}

func init() { driver.Register[dirs.Driver](factory{}) }
