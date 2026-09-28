package thread_test

import (
	"context"
	"testing"

	"github.com/lewtec/lewkit/x/driver/thread"
	_ "github.com/lewtec/lewkit/x/driver/thread/std"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	err := thread.Run(ctx, func(context.Context) error {
		thread.Do(func() {})
		cancel()
		return nil
	})
	require.NoError(t, err)
}
