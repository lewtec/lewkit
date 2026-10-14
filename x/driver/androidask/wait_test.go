package androidask

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeliverOnce(t *testing.T) {
	ch, g, err := begin()
	require.NoError(t, err)
	defer end()
	deliver(g, "ok\n")
	deliver(g, "again")
	got := <-ch
	if got != "ok\n" {
		require.FailNowf(t, "", "got %q", got)
	}
	if _, _, err := begin(); err == nil {
		require.FailNow(t, fmt.Sprint("second begin succeeded"))
	}
}
