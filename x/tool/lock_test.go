package tool

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLockSpecUsesPin(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"dependencies":[{"kind":"tool","ref":"github:getsops/sops","currentValue":"v3.10.2"}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.lock.json"), body, 0o644))
	spec, err := LockSpec(dir, "github:getsops/sops")
	require.NoError(t, err)
	require.Equal(t, "github:getsops/sops@v3.10.2", spec)
}

func TestLockSpecWalksToParent(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"dependencies":[{"kind":"tool","ref":"github:protocolbuffers/protobuf","currentValue":"v36.2"}]}`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.lock.json"), body, 0o644))
	child := filepath.Join(dir, "nested")
	require.NoError(t, os.Mkdir(child, 0o755))
	spec, err := LockSpec(child, "github:protocolbuffers/protobuf")
	require.NoError(t, err)
	require.Equal(t, "github:protocolbuffers/protobuf@v36.2", spec)
}

func TestLockSpecLatestWithoutPin(t *testing.T) {
	spec, err := LockSpec(t.TempDir(), "github:getsops/sops")
	require.NoError(t, err)
	require.Equal(t, "github:getsops/sops@latest", spec)
}

func TestLockSpecBadJSON(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "modot.lock.json"), []byte("{"), 0o644))
	_, err := LockSpec(dir, "github:getsops/sops")
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), "modot.lock.json:"))
}
