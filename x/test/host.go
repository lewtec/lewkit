package test

import (
	"context"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/stretchr/testify/require"
)

// IncompatibleOff skips when runtime.GOOS is one of hosts.
// Otherwise check must return [driver.ErrIncompatible].
// hosts are build.GOOS values.
func IncompatibleOff[T ~string](tb testing.TB, check func(context.Context) error, hosts ...T) {
	tb.Helper()
	for _, host := range hosts {
		if runtime.GOOS == string(host) {
			Skip(tb)
		}
	}
	require.ErrorIs(tb, check(tb.Context()), driver.ErrIncompatible)
}
