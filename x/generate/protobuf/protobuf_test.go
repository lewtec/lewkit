package protobuf

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunEmptyFile(t *testing.T) {
	err := Run(t.Context(), "", "")
	require.ErrorIs(t, err, errFileRequired)
}

func TestRunCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := Run(ctx, filepath.Join(t.TempDir(), "x.proto"), "")
	require.ErrorIs(t, err, context.Canceled)
}

func TestRunMissingFile(t *testing.T) {
	err := Run(t.Context(), filepath.Join(t.TempDir(), "missing.proto"), "")
	require.ErrorIs(t, err, os.ErrNotExist)
}
