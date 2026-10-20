package host

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/report"
	"github.com/stretchr/testify/require"
)

func TestSpecLoadMergesFlagsOverJSON(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"version_name": "0.1.0",
		"go_main": "."
	}`), 0o644)
	require.NoError(t, err)

	cfg, _, err := Spec{Dir: dir, ID: "br.tec.lew.flag", Version: "1.2.3"}.Load()
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.flag", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	require.Equal(t, "1.2.3", cfg.VersionName)
}

func TestGeminiExampleConfigLoads(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "examples", "app", "gemini")
	cfg, base, err := Spec{Config: dir}.Load()
	require.NoError(t, err)
	require.Equal(t, dir, base)
	require.Equal(t, "br.tec.lew.eletrocromo.gemini", cfg.PackageID)
	require.Equal(t, "Gemini", cfg.AppName)
	require.Equal(t, dir, cfg.GoMain)
}

func TestMissingConfigIsNotReported(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "app")
	_, _, err := Spec{Config: missing}.Load()
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorIs(t, err, report.ErrDontReport)
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	_, err := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	return root
}
