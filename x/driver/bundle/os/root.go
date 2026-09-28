// Package host places the stamped application under the OS homes.
package host

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/bundle"
	"github.com/lewtec/lewkit/x/driver/dirs"
)

type factory struct{}

func (factory) ID() string   { return "bundle_host" }
func (factory) Name() string { return "Application root" }
func (factory) Weight() int  { return 50 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (bundle.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Resolve(ctx context.Context, appID string) (bundle.Root, error) {
	tree, err := dirs.Resolve(ctx, appID)
	if err != nil {
		return bundle.Root{}, err
	}
	root := bundle.Root{
		ID:      appID,
		Data:    tree.Data,
		Cache:   tree.Cache,
		Config:  tree.Config,
		Profile: filepath.Join(tree.Data, "webview"),
	}
	if err := os.MkdirAll(root.Profile, 0o700); err != nil {
		return bundle.Root{}, fmt.Errorf("mkdir profile: %w", err)
	}
	return root, nil
}

func init() { driver.Register[bundle.Driver](factory{}) }
