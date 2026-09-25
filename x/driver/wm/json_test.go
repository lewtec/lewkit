package wm

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONViaCmd(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "out.json")
	err := os.WriteFile(path, []byte(`{"name":"HDMI-A-1","x":12}`), 0o644)
	require.NoError(t, err)

	got, err := JSONViaCmd[struct {
		Name string `json:"name"`
		X    int    `json:"x"`
	}](t.Context(), "cat", path)
	require.NoError(t, err)
	require.Equal(t, "HDMI-A-1", got.Name)
	require.Equal(t, 12, got.X)
}

func TestJSONViaCmdFailed(t *testing.T) {
	t.Parallel()
	_, err := JSONViaCmd[struct{}](t.Context(), "false")
	require.Error(t, err, "expected command failure")
	require.ErrorIs(t, err, ErrIPC)
	var ee *exec.ExitError
	require.ErrorAs(t, err, &ee)
}

func TestJSONViaCmdBadJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "not.json")
	err := os.WriteFile(path, []byte("not-json"), 0o644)
	require.NoError(t, err)
	_, err = JSONViaCmd[struct{}](t.Context(), "cat", path)
	require.Error(t, err, "expected decode failure")
	require.ErrorIs(t, err, ErrIPC)
	var se *json.SyntaxError
	require.ErrorAs(t, err, &se)
}
