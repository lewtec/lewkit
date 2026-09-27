package native

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver"
)

func TestDisabledWithoutEnv(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_NATIVE_TREESITTER", "")
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestEnabledNeedsLibrary(t *testing.T) {
	t.Setenv("LEWKIT_ENABLE_NATIVE_TREESITTER", "1")
	err := factory{}.CheckCompatibility(t.Context())
	if err != nil {
		require.ErrorIs(t, err, driver.ErrIncompatible)
		return
	}
	lang, ok := engine{}.Language("json")
	if !ok {
		t.Skip("libtree-sitter-json is not installed")
	}
	tree, err := lang.Parse([]byte(`{"a":1}`))
	require.NoError(t, err)
	require.NoError(t, tree.Parsed())
	root := tree.RootNode()
	require.NotEmpty(t, root.Type())
	require.NotZero(t, root.ChildCount())
}
