package llvm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLeavenCorpus(t *testing.T) {
	root := filepath.Join("testdata", "corpus")
	info, err := os.Stat(root)
	require.NoError(t, err)
	require.True(t, info.IsDir())
	var paths []string
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".ll") {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, paths)
	for _, path := range paths {
		rel, err := filepath.Rel(root, path)
		require.NoError(t, err)
		t.Run(rel, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			first, err := Take(strings.NewReader(string(raw)))
			require.NoError(t, err)
			var text strings.Builder
			require.NoError(t, first.Make(&text))
			second, err := Take(strings.NewReader(text.String()))
			require.NoError(t, err)
			var again strings.Builder
			require.NoError(t, second.Make(&again))
			require.Equal(t, text.String(), again.String())
			require.Equal(t, first, second)
		})
	}
}
