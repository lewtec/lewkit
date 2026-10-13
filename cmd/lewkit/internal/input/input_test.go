package input

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "in.bin")
	require.NoError(t, os.WriteFile(path, []byte("abc"), 0o644))
	got, err := Read(t.Context(), path)
	require.NoError(t, err)
	require.Equal(t, "abc", string(got))
}

func TestReadCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Read(ctx, "missing")
	require.Error(t, err)
}
