//go:build android

package android

import (
	"context"
	"fmt"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/lewtec/lewkit/x/ffi/jni"
	androidffi "github.com/lewtec/lewkit/x/ffi/native/android"
)

var _ notification.Driver = backend{}

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
	rc, err := jni.Int(jni.CallStatic("lewkit.Notify", "post", int32(n.ID), n.Title, n.Message, n.Urgency, n.HasProgress, percentOf(n.Progress, n.HasProgress)))
	if err != nil {
		return fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	switch rc {
	case posted:
		return nil
	case waiting:
		return wait(ctx)
	case denied:
		return fmt.Errorf("%w: notifications denied", driver.ErrUnavailable)
	default:
		return fmt.Errorf("%w: notification", driver.ErrUnavailable)
	}
}

func wait(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		rc, err := jni.Int(jni.CallStatic("lewkit.Notify", "finish"))
		if err != nil {
			return fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
		}
		switch rc {
		case posted:
			return nil
		case waiting:
		case denied:
			return fmt.Errorf("%w: notifications denied", driver.ErrUnavailable)
		default:
			return fmt.Errorf("%w: notification", driver.ErrUnavailable)
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}
