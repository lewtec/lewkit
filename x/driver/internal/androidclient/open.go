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
