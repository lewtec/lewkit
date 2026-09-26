package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusNote(t *testing.T) {
	note, ok := StatusNotification(&Metadata{Title: "Song", Artist: "Band", Length: 200, Position: 50})
	require.True(t, ok)
	require.Equal(t, "Song", note.Title)
	require.Equal(t, "Band", note.Message)
	require.Equal(t, 0.25, note.Progress)
	require.True(t, note.HasProgress)

	note, ok = StatusNotification(&Metadata{Title: "Song"})
	require.True(t, ok)
	require.Equal(t, "Unknown Artist", note.Message)

	_, ok = StatusNotification(nil)
	require.False(t, ok)
	_, ok = StatusNotification(&Metadata{})
	require.False(t, ok)
}
