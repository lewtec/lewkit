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

func TestProtocSpecUsesLockPin(t *testing.T) {
	directory := t.TempDir()
	body := []byte(`{"dependencies":[{"kind":"tool","ref":"github:protocolbuffers/protobuf","currentValue":"v36.2"}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(directory, "workspaced.lock.json"), body, 0o644))
	spec, err := protocSpec(directory)
	require.NoError(t, err)
	require.Equal(t, "github:protocolbuffers/protobuf@v36.2", spec)
}

func TestProtocSpecLatestWithoutLock(t *testing.T) {
	spec, err := protocSpec(t.TempDir())
	require.NoError(t, err)
	require.Equal(t, "github:protocolbuffers/protobuf@latest", spec)
}
