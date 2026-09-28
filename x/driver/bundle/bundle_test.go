package bundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/driver/bundle"
	_ "github.com/lewtec/lewkit/x/driver/bundle/prelude"
	_ "github.com/lewtec/lewkit/x/driver/dirs/prelude"
	"github.com/stretchr/testify/require"
)

func TestResolveUsesStampedID(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.bundle")
	data := t.TempDir()
	cache := t.TempDir()
	config := t.TempDir()
	t.Setenv("LEWKIT_DATA_DIR", data)
	t.Setenv("LEWKIT_CACHE_DIR", cache)
	t.Setenv("LEWKIT_CONFIG_DIR", config)

	root, err := bundle.Resolve(t.Context())
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.bundle", root.ID)
	require.Equal(t, data, root.Data)
	require.Equal(t, cache, root.Cache)
	require.Equal(t, config, root.Config)
	require.Equal(t, filepath.Join(data, "webview"), root.Profile)
	_, err = os.Stat(root.Profile)
	require.NoError(t, err)
	require.Equal(t, filepath.Join(cache, "share.jsonl"), bundle.SharePath(root))
}
