package dirs_test

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/dirs"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
)

func TestResolve_RejectsBadAppID(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", `a\b`, "a/../b"} {
		_, err := dirs.Resolve(t.Context(), id)
		if !errors.Is(err, dirs.ErrInvalidAppID) {
			t.Fatalf("%q: %v", id, err)
		}
	}
}
