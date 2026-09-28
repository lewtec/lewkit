//go:build linux

package webkitgtk

import (
	"testing"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/stretchr/testify/require"
)

func TestWebKitDirsUseTheSharedChain(t *testing.T) {
	t.Setenv("WEBKITGTK_LIB", "/tmp/webkit-a:/tmp/webkit-b")
	dirs := native.SearchDirs(webkitDirs()...)
	require.Equal(t, []string{"/tmp/webkit-a", "/tmp/webkit-b"}, dirs[:2])
	require.Contains(t, dirs, "/run/current-system/sw/lib")
}
