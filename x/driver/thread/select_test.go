//go:build !android

package thread_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
	_ "github.com/lewtec/lewkit/x/driver/thread/prelude"
	"github.com/stretchr/testify/require"
)

func TestJNIStaysIncompatibleOffAndroid(t *testing.T) {
	var sawJNI, sawStd bool
	for _, iface := range driver.Doctor(t.Context()) {
		for _, row := range iface.Drivers {
			switch row.ID {
			case "thread_jni":
				sawJNI = true
				require.False(t, row.Available)
				require.False(t, row.Selected)
				require.ErrorIs(t, row.Error, driver.ErrIncompatible)
			case "thread_std":
				sawStd = true
				require.True(t, row.Available)
				require.True(t, row.Selected)
			}
		}
	}
	require.True(t, sawJNI)
	require.True(t, sawStd)
}
