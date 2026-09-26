package wazero

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/driver/treesitter"
)

func TestMissingLanguage(t *testing.T) {
	_, ok := engine{}.Language("json")
	require.False(t, ok)
	require.NotContains(t, engine{}.Names(), "json")
	_, err := treesitter.Get(t.Context(), "json")
	require.ErrorIs(t, err, treesitter.ErrUnknown)
}
