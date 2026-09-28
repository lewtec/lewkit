//go:build android && cgo

package android

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/launcher"
	"github.com/lewtec/lewkit/x/entry"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

func init() { driver.Register[launcher.Confirmer](factory{}) }

func (factory) CheckCompatibility(context.Context) error {
	n, err := androidffi.JavaVMs()
	if err != nil || n < 1 {
		return fmt.Errorf("%w: no Java VM", driver.ErrIncompatible)
	}
	return nil
}

func (factory) New(context.Context) (launcher.Confirmer, error) { return backend{}, nil }

type backend struct{}

var confirmGate sync.Mutex

func (backend) Confirm(ctx context.Context, message string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	on, err := androidffi.OnLooper()
	if err != nil {
		return false, err
	}
	if on {
		return false, fmt.Errorf("%w: main looper", driver.ErrIncompatible)
	}

	confirmGate.Lock()
	defer confirmGate.Unlock()

	ch := make(chan int, 1)
	entry.HandleConfirm(func(code int) {
		select {
		case ch <- code:
		default:
		}
	})
	defer entry.HandleConfirm(nil)

	if err := androidffi.StaticVoid("confirm", "(Ljava/lang/String;)V", message); err != nil {
		return false, err
	}
	ok, err := takeConfirm(ctx, ch)
	if err != nil && ctx.Err() != nil {
		if dismissErr := androidffi.StaticVoid("dismissConfirm", "()V"); dismissErr != nil {
			return false, errors.Join(err, dismissErr)
		}
	}
	return ok, err
}

var _ launcher.Confirmer = backend{}
