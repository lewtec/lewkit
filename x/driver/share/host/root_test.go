package host

import (
	"bufio"
	"encoding/json"
	"os"
	"testing"

	"github.com/lewtec/lewkit/x/driver/bundle"
	"github.com/lewtec/lewkit/x/driver/share"
	"github.com/stretchr/testify/require"
)

func TestOutWritesDrop(t *testing.T) {
	t.Setenv("LEWKIT_APP_ID", "br.tec.lew.share")
	t.Setenv("LEWKIT_DATA_DIR", t.TempDir())
	t.Setenv("LEWKIT_CACHE_DIR", t.TempDir())
	t.Setenv("LEWKIT_CONFIG_DIR", t.TempDir())

	err := backend{}.Out(t.Context(), share.Item{Text: "hello"})
	require.NoError(t, err)

	root, err := bundle.Resolve(t.Context())
	require.NoError(t, err)
	file, err := os.Open(bundle.SharePath(root))
	require.NoError(t, err)
	defer file.Close()
	var got struct {
		Text string `json:"text"`
	}
	require.NoError(t, json.NewDecoder(bufio.NewReader(file)).Decode(&got))
	require.Equal(t, "hello", got.Text)
}
