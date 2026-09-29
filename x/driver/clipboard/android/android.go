//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"image"

	"github.com/lewtec/lewkit/x/driver"
	host "github.com/lewtec/lewkit/x/driver/android"
	"github.com/lewtec/lewkit/x/driver/clipboard"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[clipboard.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "clipboard_android" }
func (factory) Name() string { return "Android clipboard" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (clipboard.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) WriteText(ctx context.Context, text string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	app, err := host.Context()
	if err != nil {
		return err
	}
	defer app.Release()
	service, err := host.Text(jni.StaticField("android.content.Context", "CLIPBOARD_SERVICE"))
	if err != nil {
		return err
	}
	manager, err := host.Ref(app.Call("getSystemService", service))
	if err != nil {
		return err
	}
	if manager == nil {
		return fmt.Errorf("%w: clipboard", driver.ErrUnavailable)
	}
	defer manager.Release()
	clip, err := host.Ref(jni.CallStatic("android.content.ClipData", "newPlainText", "text", text))
	if err != nil {
		return err
	}
	if clip == nil {
		return fmt.Errorf("%w: clipboard", driver.ErrUnavailable)
	}
	defer clip.Release()
	_, err = manager.Call("setPrimaryClip", clip)
	return err
}

func (backend) WriteImage(ctx context.Context, _ image.Image) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%w: image clipboard", driver.ErrIncompatible)
}
