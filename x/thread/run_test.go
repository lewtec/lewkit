package thread

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	err := Run(ctx, func(context.Context) error {
		Do(func() {})
		cancel()
		return nil
	})
	require.NoError(t, err)
}
