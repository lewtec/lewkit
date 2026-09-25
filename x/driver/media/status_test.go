package media

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusNote(t *testing.T) {
	note, ok := statusNote(&Metadata{Title: "Song", Artist: "Band", Length: 200, Position: 50})
	require.True(t, ok)
	require.Equal(t, "Song", note.Title)
	require.Equal(t, "Band", note.Message)
	require.Equal(t, 0.25, note.Progress)
	require.True(t, note.HasProgress)

	note, ok = statusNote(&Metadata{Title: "Song"})
	require.True(t, ok)
	require.Equal(t, "Unknown Artist", note.Message)

	_, ok = statusNote(nil)
	require.False(t, ok)
	_, ok = statusNote(&Metadata{})
	require.False(t, ok)
}

func TestRunActionUnknown(t *testing.T) {
	err := RunAction(t.Context(), "nope")
	require.EqualError(t, err, "unknown action: nope")
}
