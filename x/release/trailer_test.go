package release

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrailerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Demo.AppImage")
	require.NoError(t, os.WriteFile(path, []byte("\x7fELF-demo"), 0o755))
	if _, err := OpenTrailer(path); err == nil {
		require.FailNow(t, fmt.Sprint("plain executable opened as a trailer"))
	}
	err := AppendTrailer(path, map[string][]byte{
		MarkerFile: []byte("br.tec.lew.demo\n"),
		"icon.png": []byte("png"),
	})
	require.NoError(t, err)
	tr, err := OpenTrailer(path)
	require.NoError(t, err)
	defer tr.Close()
	id, err := tr.Bytes(MarkerFile)
	require.NoError(t, err)
	if string(id) != "br.tec.lew.demo\n" {
		require.FailNowf(t, "", "id %q", id)
	}
	icon, err := tr.Bytes("icon.png")
	if err != nil || string(icon) != "png" {
		require.FailNowf(t, "", "icon %q %v", icon, err)
	}
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	if string(raw[:4]) != "\x7fELF" {
		require.FailNowf(t, "", "prefix %q", raw[:4])
	}
}
