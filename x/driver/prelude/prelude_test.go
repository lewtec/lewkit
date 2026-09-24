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
	out := t.TempDir()
	dest := filepath.Join(out, "prelude", "prelude.go")
	require.NoError(t, gprelude.Run(t.Context(), dir, dest))
	err = filepath.WalkDir(out, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || entry.Name() != "prelude.go" {
			return nil
		}
		rel, err := filepath.Rel(out, name)
		if err != nil {
			return err
		}
		got, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		want, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return err
		}
		assert.Equal(t, string(want), string(got), rel)
		return nil
	})
	require.NoError(t, err)
}
