package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/dirs"
)

func init() { driver.Register[dirs.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "dirs_android" }
func (factory) Name() string { return "Android directories" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	_, err := host.Open(ctx)
	return err
}

func (factory) New(context.Context) (dirs.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Resolve(ctx context.Context, appID string) (dirs.Dirs, error) {
	client, err := host.Open(ctx)
	if err != nil {
		return dirs.Dirs{}, err
	}
	dataDir, err := client.DataDir(ctx, appID)
	if err != nil {
		return dirs.Dirs{}, err
	}
	got := pathsFromDataDir(dataDir)
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs.Dirs{}, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return got, nil
}

func pathsFromDataDir(dataDir string) dirs.Dirs {
	data := filepath.Join(dataDir, "files")
	cache := filepath.Join(dataDir, "cache")
	return dirs.Dirs{
		Data:   data,
		Cache:  cache,
		Config: filepath.Join(data, "config"),
		Inbox:  filepath.Join(cache, "inbox"),
	}
}
