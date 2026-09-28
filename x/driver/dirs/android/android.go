//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/dirs"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[dirs.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "dirs_android" }
func (factory) Name() string { return "Android directories" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (dirs.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Resolve(_ context.Context, _ string) (dirs.Dirs, error) {
	data, err := androidffi.StaticString("dataDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	cache, err := androidffi.StaticString("cacheDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	config, err := androidffi.StaticString("configDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	got := dirs.Dirs{
		Data:   data,
		Cache:  cache,
		Config: config,
		Inbox:  filepath.Join(cache, "inbox"),
	}
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs.Dirs{}, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return got, nil
}
