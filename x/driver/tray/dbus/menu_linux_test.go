//go:build linux

package dbus

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/stretchr/testify/require"
)

func TestLayoutNestedMenu(t *testing.T) {
	nodes := indexLayout(tray.Tree([]tray.Item{
		{Label: "Open"},
		{Label: "More", Children: []tray.Item{{Label: "About"}, {Separator: true}}},
		{Label: "Quit", Disabled: true, Checked: true},
	}))
	root := nodes[0]
	require.Len(t, root.Children, 3)
	about := nodes[3]
	require.Equal(t, "About", about.Properties["label"].Value())
	separator := nodes[4]
	require.Equal(t, "separator", separator.Properties["type"].Value())
	quit := nodes[5]
	require.Equal(t, false, quit.Properties["enabled"].Value())
	require.Equal(t, int32(1), quit.Properties["toggle-state"].Value())
}
