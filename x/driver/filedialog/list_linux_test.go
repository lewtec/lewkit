//go:build linux

package filedialog_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/filedialog"
	_ "github.com/lewtec/lewkit/x/driver/filedialog/gtk"
	_ "github.com/lewtec/lewkit/x/driver/filedialog/qt"
	"github.com/stretchr/testify/require"
)

func TestLinuxChooser(t *testing.T) {
	t.Setenv("XDG_CURRENT_DESKTOP", "")
	t.Setenv("XDG_SESSION_DESKTOP", "")
	handles, err := driver.List[filedialog.Driver](t.Context())
	if err != nil {
		require.ErrorIs(t, err, driver.ErrUnavailable)
		return
	}
	require.NotEmpty(t, handles)
	require.Contains(t, []string{"filedialog_gtk", "filedialog_qt"}, handles[0].ID)
}
