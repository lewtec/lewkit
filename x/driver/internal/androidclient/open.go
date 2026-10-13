// Package androidclient opens the process binder client for an Android driver.
// A binder failure is [driver.ErrIncompatible]. The client itself stays in x/ffi/android.
package androidclient

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	ffiandroid "github.com/lewtec/lewkit/x/ffi/android"
)

// Open returns the process binder client.
func Open(ctx context.Context) (*ffiandroid.Client, error) {
	client, err := ffiandroid.ForAndroid(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return client, nil
}

// Interactive opens the binder client and reports whether the session is interactive.
// A binder or query failure is [driver.ErrIncompatible].
func Interactive(ctx context.Context) error {
	client, err := Open(ctx)
	if err != nil {
		return err
	}
	if _, err := client.Interactive(ctx); err != nil {
		return fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return nil
}
