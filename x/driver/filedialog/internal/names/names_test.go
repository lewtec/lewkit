package names

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTake(t *testing.T) {
	seen := map[string]struct{}{}
	require.Equal(t, "a_b.txt", Take("a/b.txt", seen))
	require.Equal(t, "a_b (2).txt", Take("a/b.txt", seen))
	require.Equal(t, "_", Take(".", seen))
	require.Equal(t, "a_b", Clean(`a\b`))
}
