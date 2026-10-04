package osascript

import (
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/notification"
	"github.com/stretchr/testify/require"
)

func TestScript(t *testing.T) {
	got := script(notification.Notification{Title: `say "hi"`, Message: "line\n2", Urgency: "critical"})
	require.Equal(t, `display notification "line 2" with title "say \"hi\"" sound name "Basso"`, got)
	require.Contains(t, script(notification.Notification{Message: "done"}), `with title "Notification"`)
}

func TestNotDarwin(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip()
	}
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}
