package webview

import (
	"testing"

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

func TestContentType(t *testing.T) {
	require.Equal(t, "text/javascript", ContentType("app.JS"))
	require.Equal(t, "text/html", ContentType("index.html"))
	require.Equal(t, "application/octet-stream", ContentType("blob"))
}
