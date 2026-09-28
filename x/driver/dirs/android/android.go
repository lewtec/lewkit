//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/dirs"
	"github.com/lewtec/lewkit/x/ffi/jni"
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
	data, err := hostString("dataDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	cache, err := hostString("cacheDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	config, err := hostString("configDir")
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

var errHostValue = errors.New("lewkit.Host value")

func hostString(name string) (string, error) {
	v, err := jni.CallStatic("lewkit.Host", name)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%w: %s returned %T", errHostValue, name, v)
	}
	return s, nil
}
