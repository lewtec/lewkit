package taskgroup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWithoutSession(t *testing.T) {
	sess, ctx := New(t.Context(), DefaultLimits())
	detached := WithoutSession(ctx)
	if FromContext(detached) != nil {
		require.FailNow(t, fmt.Sprint("session leaked"))
	}
	if FromContext(ctx) != sess {
		require.FailNow(t, fmt.Sprint("parent session"))
	}
	require.NoError(t, sess.Wait())
}
