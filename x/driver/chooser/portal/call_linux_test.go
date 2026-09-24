//go:build linux

package portal

import (
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver/chooser"
	"github.com/stretchr/testify/require"
)

func TestPathFromURI(t *testing.T) {
	got, err := pathFromURI("file:///tmp/My%20Song.mp3")
	require.NoError(t, err)
	require.Equal(t, "/tmp/My Song.mp3", got)

	got, err = pathFromURI("file://localhost/tmp/a")
	require.NoError(t, err)
	require.Equal(t, "/tmp/a", got)

	_, err = pathFromURI("https://example.com/a")
	require.Error(t, err)
}

func TestOptions(t *testing.T) {
	got := options(chooser.Request{
		Directory: "/tmp",
		Name:      "note.txt",
		Save:      true,
		Filters: []chooser.Filter{{
			Name:     "Text",
			Patterns: []string{"*.txt", ""},
		}},
	})
	require.Equal(t, "b", got["modal"].Signature().String())
	require.Equal(t, "ay", got["current_folder"].Signature().String())
	require.Equal(t, []byte("/tmp\x00"), got["current_folder"].Value())
	require.Equal(t, "note.txt", got["current_name"].Value())
	require.Equal(t, "a(sa(us))", got["filters"].Signature().String())
	require.False(t, got["multiple"].Value().(bool))
	require.False(t, got["directory"].Value().(bool))
}

func TestPathsFromResults(t *testing.T) {
	_, err := pathsFromResults(responseCancel, nil)
	require.ErrorIs(t, err, chooser.ErrCanceled)

	_, err = pathsFromResults(2, nil)
	require.Error(t, err)

	got, err := pathsFromResults(responseSuccess, map[string]dbus.Variant{
		"uris": dbus.MakeVariant([]string{"file:///tmp/a", "file:///tmp/b"}),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/a", "/tmp/b"}, got)
}
