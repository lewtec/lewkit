//go:build android

package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/opener"
	"github.com/lewtec/lewkit/x/ffi/jni"
)

const (
	opened  = 0
	bad     = 1
	missing = 2
	outside = 3
)

var _ opener.Driver = backend{}

func (factory) CheckCompatibility(context.Context) error {
	return driver.JavaVM()
}

func (factory) New(context.Context) (opener.Driver, error) { return backend{}, nil }

type backend struct{}

func (backend) Open(ctx context.Context, target string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	rc, err := jni.Int(jni.CallStatic("lewkit.Open", "open", target))
	if err != nil {
		return fmt.Errorf("%w: %w", driver.ErrUnavailable, err)
	}
	switch rc {
	case opened:
		return nil
	case bad:
		return fmt.Errorf("%w: bad target", driver.ErrUnavailable)
	case outside:
		return fmt.Errorf("%w: file is outside shared paths", driver.ErrUnavailable)
	case missing:
		return fmt.Errorf("%w: open", driver.ErrUnavailable)
	default:
		return fmt.Errorf("%w: open", driver.ErrUnavailable)
	}
}
