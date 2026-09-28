//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

var errNotify = errors.New("android notify")

func init() { driver.Register[notification.Driver](factory{}) }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (notification.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Notify(ctx context.Context, n notification.Notification) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	msg, err := androidffi.StaticStrings("notify", notifySig, notifyArgs(n)...)
	if err != nil {
		return fmt.Errorf("android notify: %w", err)
	}
	if msg != "" {
		return fmt.Errorf("%w: %s", errNotify, msg)
	}
	return nil
}
