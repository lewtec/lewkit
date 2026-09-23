//go:build linux

package webkitgtk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLibraryDirectoriesIncludeNixOSSystem(t *testing.T) {
	t.Setenv("WEBKITGTK_LIB", "/tmp/webkit-a:/tmp/webkit-b")
	dirs := libraryDirectories()
	require.Equal(t, []string{"/tmp/webkit-a", "/tmp/webkit-b", nixOSSystemLib}, dirs)
}
