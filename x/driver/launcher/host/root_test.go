package host

import (
	"context"
	"fmt"
	"testing"

	"github.com/lewtec/lewkit/x/driver/askwire"
	"github.com/lewtec/lewkit/x/driver/hostask"
	"github.com/stretchr/testify/require"
)

func TestNeedsHost(t *testing.T) {
	t.Setenv("ELETROCROMO_ASK_DIR", "")
	t.Setenv("ELETROCROMO_NO_UI", "")
	t.Setenv("LEWKIT_NO_UI", "")
	t.Setenv("ELETROCROMO_CACHE_DIR", "")
	t.Setenv("LEWKIT_CACHE_DIR", "")
	hostask.Set(nil)
	t.Cleanup(func() { hostask.Set(nil) })
	if err := (base{}).CheckCompatibility(t.Context()); err == nil {
		require.FailNow(t, fmt.Sprint("host launcher is compatible without an ask host"))
	}
	hostask.Set(func(context.Context, int, string, string) (string, error) {
		return askwire.Format(askwire.StatusYes, ""), nil
	})
	require.NoError(t, (base{}).CheckCompatibility(t.Context()))
	ok, err := backend{}.Confirm(t.Context(), "Delete?")
	if err != nil || !ok {
		require.FailNowf(t, "", "confirm %v %v", ok, err)
	}
}
