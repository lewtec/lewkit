package tray

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMenuTreeIDs(t *testing.T) {
	nodes := Tree([]Item{
		{Label: "Open"},
		{Label: "More", Children: []Item{{Label: "About"}, {Separator: true}}},
		{Label: "Quit"},
	})
	require.Equal(t, 1, nodes[0].ID)
	require.Equal(t, 2, nodes[1].ID)
	require.Equal(t, 3, nodes[1].Children[0].ID)
	require.Equal(t, 4, nodes[1].Children[1].ID)
	require.Equal(t, 5, nodes[2].ID)
	item, ok := Find(nodes, 3)
	require.True(t, ok)
	require.Equal(t, "About", item.Label)
	_, ok = Find(nodes, 9)
	require.False(t, ok)
}

func TestNameOnlyIcon(t *testing.T) {
	got, err := Source(Icon{Name: "applications-system"})
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestRejectUnknownBytes(t *testing.T) {
	_, err := Source(Icon{Bytes: []byte("not an image")})
	require.ErrorIs(t, err, ErrIcon)
}
