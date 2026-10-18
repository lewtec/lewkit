package android

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/lewtec/lewkit/x/driver/internal/androidclient"
)

var (
	ErrBackendResolveMkdir = errors.New("mkdir")
)

func init() { driver.Register[dirs.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "dirs_android" }
func (factory) Name() string { return "Android directories" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	_, err := androidclient.Open(ctx)
	return err
}

func (factory) New(context.Context) (dirs.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Resolve(ctx context.Context, appID string) (dirs.Dirs, error) {
	client, err := androidclient.Open(ctx)
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
			return dirs.Dirs{}, fmt.Errorf("%w %s: %w", ErrBackendResolveMkdir, dir, err)
		}
	}
	return got, nil
}

func pathsFromDataDir(dataDir string) dirs.Dirs {
	data := path.Join(dataDir, "files")
	cache := path.Join(dataDir, "cache")
	return dirs.Dirs{
		Data:   data,
		Cache:  cache,
		Config: path.Join(data, "config"),
		Inbox:  path.Join(cache, "inbox"),
	}
}
