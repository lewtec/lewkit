package android

import (
	"encoding/json"
	"testing"

	"github.com/lewtec/lewkit/x/driver/filedialog"
	"github.com/stretchr/testify/require"
)

func TestEncodeRequest(t *testing.T) {
	raw, err := encodeRequest(filedialog.Request{
		Title:     "Music",
		Directory: "/storage/emulated/0/Music",
		Name:      "a.mp3",
		Filters:   []filedialog.Filter{{Patterns: []string{"*.mp3", "*", "a/b"}}},
		Multiple:  true,
		Folder:    true,
	})
	require.NoError(t, err)
	var got requestJSON
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	require.Equal(t, requestJSON{
		Title:      "Music",
		Directory:  "/storage/emulated/0/Music",
		Name:       "a.mp3",
		Extensions: []string{"mp3"},
		Multiple:   true,
		Folder:     true,
	}, got)
}

func TestEncodeDefaultTitle(t *testing.T) {
	raw, err := encodeRequest(filedialog.Request{Save: true, Name: "note.txt"})
	require.NoError(t, err)
	var got requestJSON
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	require.Equal(t, "Save", got.Title)
	require.True(t, got.Save)
	require.False(t, got.Folder)
}

func TestDecodeOutcome(t *testing.T) {
	paths, err := decodeOutcome(`{"paths":["/tmp/a","/tmp/b"]}`)
	require.NoError(t, err)
	require.Equal(t, []string{"/tmp/a", "/tmp/b"}, paths)

	_, err = decodeOutcome(`{"canceled":true}`)
	require.ErrorIs(t, err, filedialog.ErrCanceled)

	_, err = decodeOutcome(`{"error":"no activity"}`)
	require.ErrorIs(t, err, errDialog)
	require.ErrorContains(t, err, "no activity")

	_, err = decodeOutcome(`{"paths":[]}`)
	require.ErrorIs(t, err, errDialog)

	_, err = decodeOutcome(`not json`)
	require.ErrorIs(t, err, errDialog)
}

func TestFactoryID(t *testing.T) {
	require.Equal(t, "filedialog_android", factory{}.ID())
	require.Equal(t, "Documents", factory{}.Name())
	require.Equal(t, 80, factory{}.Weight())
}
