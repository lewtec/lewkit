package dirs_test

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/dirs"
	_ "github.com/lewtec/lewkit/x/driver/dirs/os"
)

func TestResolve_RejectsBadAppID(t *testing.T) {
	for _, id := range []string{"", "..", "a/b", `a\b`, "a/../b"} {
		_, err := dirs.Resolve(context.Background(), id)
		if !errors.Is(err, dirs.ErrInvalidAppID) {
			t.Fatalf("%q: %v", id, err)
		}
	}
}
