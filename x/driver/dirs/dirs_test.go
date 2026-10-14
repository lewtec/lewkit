package dirs_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/dirs"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
	"github.com/stretchr/testify/require"
)

func TestResolve_RejectsBadAppID(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", `a\b`, "a/../b"} {
		_, err := dirs.Resolve(t.Context(), id)
		require.ErrorIs(t, err, dirs.ErrInvalidAppID)
	}
}
