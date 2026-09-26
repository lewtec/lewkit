package webview

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	err := Config{}.Validate()
	require.ErrorIs(t, err, ErrPage)
	err = Config{Width: -1, HTML: "<p></p>"}.Validate()
	require.ErrorIs(t, err, ErrSize)
	require.NoError(t, Config{HTML: "<p></p>"}.Validate())
}

func TestAssetName(t *testing.T) {
	name, err := AssetName("/")
	require.NoError(t, err)
	require.Equal(t, "index.html", name)
	name, err = AssetName("/js/app.js")
	require.NoError(t, err)
	require.Equal(t, "js/app.js", name)
	_, err = AssetName("/../secret")
	require.NoError(t, err)
	name, err = AssetName("/../secret")
	require.NoError(t, err)
	require.Equal(t, "secret", name)
}

func TestReadPage(t *testing.T) {
	files := fstest.MapFS{
		"index.html": {Data: []byte("file")},
		"app.js":     {Data: []byte("js")},
	}
	body, kind, err := ReadPage("<p>hi</p>", files, "/")
	require.NoError(t, err)
	require.Equal(t, "text/html", kind)
	require.Equal(t, "<p>hi</p>", string(body))

	body, kind, err = ReadPage("  ", files, "/index.html")
	require.NoError(t, err)
	require.Equal(t, "text/html", kind)
	require.Equal(t, "file", string(body))

	body, kind, err = ReadPage("", files, "/app.js")
	require.NoError(t, err)
	require.Equal(t, "text/javascript", kind)
	require.Equal(t, "js", string(body))

	_, _, err = ReadPage("", nil, "/app.js")
	require.ErrorIs(t, err, ErrAsset)

	_, _, err = ReadPage("", files, "/missing")
	require.Error(t, err)
}

func TestContentType(t *testing.T) {
	require.Equal(t, "text/javascript", ContentType("app.JS"))
	require.Equal(t, "text/html", ContentType("index.html"))
	require.Equal(t, "application/octet-stream", ContentType("blob"))
}
