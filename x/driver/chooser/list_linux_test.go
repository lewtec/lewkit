//go:build linux

package chooser_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/chooser"
	_ "github.com/lewtec/lewkit/x/driver/chooser/gtk"
	_ "github.com/lewtec/lewkit/x/driver/chooser/qt"
	"github.com/stretchr/testify/require"
)

func TestLinuxChooser(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	t.Setenv("XDG_SESSION_DESKTOP", "")
	handles, err := driver.List[chooser.Driver](t.Context())
	if err != nil {
		require.ErrorIs(t, err, driver.ErrUnavailable)
		return
	}
	require.NotEmpty(t, handles)
	require.Contains(t, []string{"chooser_gtk", "chooser_qt"}, handles[0].ID)
}
