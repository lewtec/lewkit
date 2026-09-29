//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
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

func (backend) Resolve(ctx context.Context, _ string) (dirs.Dirs, error) {
	if err := ctx.Err(); err != nil {
		return dirs.Dirs{}, err
	}
	app, err := host.Context()
	if err != nil {
		return dirs.Dirs{}, err
	}
	defer app.Release()
	data, err := dirPath(app, "getFilesDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	cache, err := dirPath(app, "getCacheDir")
	if err != nil {
		return dirs.Dirs{}, err
	}
	got := dirs.Dirs{
		Data:   data,
		Cache:  cache,
		Config: filepath.Join(data, "config"),
		Inbox:  filepath.Join(cache, "inbox"),
	}
	for _, dir := range []string{got.Data, got.Cache, got.Config, got.Inbox} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return dirs.Dirs{}, fmt.Errorf("mkdir %s: %w", dir, err)
		}
	}
	return got, nil
}

func dirPath(app *jni.Ref, method string) (string, error) {
	file, err := host.Ref(app.Call(method))
	if err != nil {
		return "", err
	}
	if file == nil {
		return "", fmt.Errorf("%w: %s", driver.ErrUnavailable, method)
	}
	defer file.Release()
	text, err := host.Text(file.Call("getAbsolutePath"))
	if err != nil {
		return "", err
	}
	if text == "" {
		return "", fmt.Errorf("%w: %s", driver.ErrUnavailable, method)
	}
	return text, nil
}
