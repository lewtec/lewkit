//go:build android && cgo

package android

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[filedialog.Driver](factory{}) }

type factory struct{}

func (factory) ID() string   { return "filedialog_android" }
func (factory) Name() string { return "Documents" }
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

func (backend) Choose(ctx context.Context, req filedialog.Request) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", filedialog.ErrRequest, err)
	}
	if on, err := androidffi.OnLooper(); err == nil && on {
		return nil, fmt.Errorf("%w: main thread", errDialog)
	}
	spec, err := encodeRequest(req)
	if err != nil {
		return nil, err
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			if err := androidffi.StaticVoid("cancelChoose", "()V"); err != nil {
				slog.Error("file dialog cancel", "err", err)
			}
		case <-done:
		}
	}()
	raw, err := androidffi.CallString("chooseFile", spec)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return decodeOutcome(raw)
}

var _ driver.DriverFactory[filedialog.Driver] = factory{}
var _ driver.Weighter = factory{}
