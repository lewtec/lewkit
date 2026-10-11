//go:build android

package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[filedialog.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "filedialog_android" }
func (factory) Name() string { return "Android file chooser" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (filedialog.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Choose(ctx context.Context, req filedialog.Request) (paths []string, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	if on, err := androidffi.OnLooper(); err == nil && on {
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, errMainLooper)
	}
	ch, gen, err := beginPick()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	defer endPick()

	// The callback runs on the main looper. It must not call back into JNI:
	// Choose may be blocking the loader thread on ch.
	proxy, err := jni.Proxy("java.util.function.BiConsumer", func(method string, args []any) (any, error) {
		if method != "accept" {
			return nil, nil
		}
		status, paths, convErr := listenerArgs(args)
		if convErr != nil {
			deliverPick(gen, "bad result", "")
			return nil, nil
		}
		deliverPick(gen, status, paths)
		return nil, nil
	})
	if err != nil {
		return nil, err
	}
	defer proxy.Release()
	if _, err = jni.CallStatic("lewkit.FileChooser", "setListener", proxy); err != nil {
		return nil, err
	}
	defer func() {
		if _, clearErr := jni.CallStatic("lewkit.FileChooser", "setListener", nil); clearErr != nil {
			err = errors.Join(err, clearErr)
		}
	}()
	if _, err = jni.CallStatic(
		"lewkit.FileChooser",
		"open",
		req.TitleOrDefault(),
		req.Directory,
		req.Name,
		extensionList(req.Filters),
		req.Multiple,
		req.Folder,
		req.Save,
	); err != nil {
		return nil, err
	}
	select {
	case got := <-ch:
		return resultOf(got.status, got.paths)
	case <-ctx.Done():
		if _, err = jni.CallStatic("lewkit.FileChooser", "cancel"); err != nil {
			return nil, fmt.Errorf("%w: %w", ctx.Err(), err)
		}
		return nil, ctx.Err()
	}
}
