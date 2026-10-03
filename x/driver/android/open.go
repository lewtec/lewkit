package android

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	ffiandroid "github.com/lewtec/lewkit/x/ffi/android"
)

// Open returns the process binder client.
// A process that is not Android is [driver.ErrIncompatible].
func Open(ctx context.Context) (*ffiandroid.Client, error) {
	client, err := ffiandroid.ForAndroid(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", driver.ErrIncompatible, err)
	}
	return client, nil
}
