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
func IncompatibleOff(tb testing.TB, check func(context.Context) error, hosts ...string) {
	tb.Helper()
	for _, host := range hosts {
		if runtime.GOOS == host {
			Skip(tb)
		}
	}
	require.ErrorIs(tb, check(tb.Context()), driver.ErrIncompatible)
}
