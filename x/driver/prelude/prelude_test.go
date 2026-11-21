package prelude

import (
	"os"
	"path/filepath"
	"testing"

	gprelude "github.com/lewtec/lewkit/x/generate/prelude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedMatches(t *testing.T) {
	dir, err := filepath.Abs("..")
	require.NoError(t, err)
	dest := filepath.Join(t.TempDir(), "prelude.go")
	require.NoError(t, gprelude.Run(t.Context(), dir, dest))
	got, err := os.ReadFile(dest)
	require.NoError(t, err)
	want, err := os.ReadFile("prelude.go")
	require.NoError(t, err)
	assert.Equal(t, string(want), string(got))
}
