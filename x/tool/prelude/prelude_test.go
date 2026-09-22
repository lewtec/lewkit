package prelude_test

import (
	"testing"

	"github.com/lewtec/lewkit/x/tool"
	_ "github.com/lewtec/lewkit/x/tool/prelude"
	"github.com/lewtec/lewkit/x/tool/registry"

	"github.com/stretchr/testify/require"
)

func TestPreludeRegistersBackends(t *testing.T) {
	for _, id := range []string{"github", "mise", "registry"} {
		backend, err := tool.Get(id)
		require.NoError(t, err)
		require.NotEmpty(t, backend.Name())
	}
	names := registry.ListTools()
	require.Contains(t, names, "uv")
	require.Contains(t, names, "golang")
	require.Contains(t, names, "protobuf")
}
