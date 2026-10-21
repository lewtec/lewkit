package astro

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssemble(t *testing.T) {
	var provider Provider
	t.Run("server", func(t *testing.T) {
		project := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "server"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "server", "entry.mjs"), []byte("export default {}\n"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "client"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "client", "app.js"), []byte("x"), 0o644))
		worker := filepath.Join(t.TempDir(), "worker")
		require.NoError(t, provider.assemble(project, worker))
		_, err := os.Stat(filepath.Join(worker, "entry.mjs"))
		require.NoError(t, err)
		body, err := os.ReadFile(filepath.Join(worker, "assets", "app.js"))
		require.NoError(t, err)
		require.Equal(t, "x", string(body))
	})

	t.Run("legacy", func(t *testing.T) {
		project := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "_worker.js"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "_worker.js", "index.js"), []byte("export default {}\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "app.css"), []byte("body{}\n"), 0o644))
		worker := filepath.Join(t.TempDir(), "worker")
		require.NoError(t, provider.assemble(project, worker))
		_, err := os.Stat(filepath.Join(worker, "entry.mjs"))
		require.NoError(t, err)
		body, err := os.ReadFile(filepath.Join(worker, "assets", "app.css"))
		require.NoError(t, err)
		require.Equal(t, "body{}\n", string(body))
	})
}
