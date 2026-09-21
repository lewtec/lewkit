package fs

import (
	iofs "io/fs"
	"testing"
	"testing/fstest"

	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/require"
)

func TestStripTopDirectoryThenCopy(t *testing.T) {
	t.Parallel()
	source := fstest.MapFS{
		"demo-1.0.0":          {Mode: iofs.ModeDir},
		"demo-1.0.0/bin":      {Mode: iofs.ModeDir},
		"demo-1.0.0/bin/demo": {Data: []byte("payload"), Mode: 0o755},
		"demo-1.0.0/README":   {Data: []byte("hi")},
	}
	destination := copyStripped(t, source)
	body, err := path.New("bin", "demo").ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "payload", string(body))
	body, err = path.New("README").ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "hi", string(body))
	exists, err := path.New("demo-1.0.0").Exists(destination)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestStripTopDirectoryLeavesMixedTree(t *testing.T) {
	t.Parallel()
	source := fstest.MapFS{
		"keep":  {Mode: iofs.ModeDir},
		"notes": {Data: []byte("hi")},
	}
	destination := copyStripped(t, source)
	exists, err := path.New("keep").IsDir(destination)
	require.NoError(t, err)
	require.True(t, exists)
	body, err := path.New("notes").ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "hi", string(body))
}

func TestStripTopDirectoryLeavesSingleFile(t *testing.T) {
	t.Parallel()
	source := fstest.MapFS{
		"codex": {Data: []byte("bin"), Mode: 0o755},
	}
	destination := copyStripped(t, source)
	body, err := path.New("codex").ReadFile(destination)
	require.NoError(t, err)
	require.Equal(t, "bin", string(body))
}

func copyStripped(t *testing.T, source iofs.FS) *path.Root {
	t.Helper()
	root, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, root)
	require.NoError(t, path.New("out").MkdirAll(root, 0o755))
	destination, err := path.New("out").OpenRoot(root)
	require.NoError(t, err)
	test.CloseOnCleanup(t, destination)
	require.NoError(t, Copy(t.Context(), destination, StripTopDirectory(t.Context(), source)))
	return destination
}
