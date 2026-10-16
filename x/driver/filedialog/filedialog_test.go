package filedialog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	require.NoError(t, Request{}.Validate())
	require.NoError(t, Request{Folder: true, Multiple: true}.Validate())
	require.NoError(t, Request{Save: true, Name: "a.txt"}.Validate())
	require.Error(t, Request{Save: true, Folder: true}.Validate())
	require.Error(t, Request{Save: true, Multiple: true}.Validate())
}

func TestTitleOrDefault(t *testing.T) {
	require.Equal(t, "Notes", Request{Title: "Notes"}.TitleOrDefault())
	require.Equal(t, "Open", Request{}.TitleOrDefault())
	require.Equal(t, "Choose folder", Request{Folder: true}.TitleOrDefault())
	require.Equal(t, "Save", Request{Save: true}.TitleOrDefault())
}

func TestExtensions(t *testing.T) {
	got := Extensions([]Filter{{Patterns: []string{"*.mp3", ".flac", "*", "a/b", ""}}})
	require.Equal(t, []string{"mp3", "flac"}, got)
}

func TestChooseRejectsRequest(t *testing.T) {
	_, err := Choose(t.Context(), Request{Save: true, Folder: true})
	require.ErrorIs(t, err, ErrRequest)
}

func TestAccept(t *testing.T) {
	require.NoError(t, Accept(t.Context(), Request{}))
	err := Accept(t.Context(), Request{Save: true, Folder: true})
	require.ErrorIs(t, err, ErrRequest)
	require.EqualError(t, err, "file dialog request: save a folder")

	ctx := t.Context()
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	err = Accept(ctx, Request{})
	require.ErrorIs(t, err, context.Canceled)
}
