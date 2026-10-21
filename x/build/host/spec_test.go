package host

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/report"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
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

	cfg, _, err := Spec{Dir: dir, ID: "br.tec.lew.flag", Version: "1.2.3"}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.flag", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	require.Equal(t, "1.2.3", cfg.VersionName)
}

func TestGeminiExampleConfigLoads(t *testing.T) {
	dir := filepath.Join(moduleRoot(t), "examples", "app", "gemini")
	cfg, base, err := Spec{Config: dir}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, dir, base)
	require.Equal(t, "br.tec.lew.eletrocromo.gemini", cfg.PackageID)
	require.Equal(t, "Gemini", cfg.AppName)
	require.Equal(t, dir, cfg.GoMain)
}

func TestSpecIgnoresJSONVersion(t *testing.T) {
	dir := t.TempDir()
	git(t, dir, "init", "-b", "main")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README"), []byte("a\n"), 0o644))
	git(t, dir, "add", "README")
	git(t, dir, "commit", "-m", "init")
	git(t, dir, "tag", "v0.2.0")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"version_name": "9.9.9",
		"version_code": 99,
		"go_main": "."
	}`), 0o644))

	cfg, _, err := Spec{Dir: dir, Config: filepath.Join(dir, "eletrocromo.json")}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "0.2.0", cfg.VersionName)
	require.Equal(t, 1, cfg.VersionCode)
}

func TestSpecMainFlagBeatsEntry(t *testing.T) {
	dir := t.TempDir()
	main := filepath.Join(dir, "cmd")
	require.NoError(t, os.Mkdir(main, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "static:missing.zip",
		"go_main": "."
	}`), 0o644))

	cfg, _, err := Spec{Dir: dir, Config: filepath.Join(dir, "eletrocromo.json"), GoMain: main}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, main, cfg.GoMain)
}

func TestSpecEntryBeatsGoMain(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other")
	require.NoError(t, os.Mkdir(other, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:other",
		"go_main": "."
	}`), 0o644))

	cfg, _, err := Spec{Dir: dir, Config: filepath.Join(dir, "eletrocromo.json")}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, other, cfg.GoMain)
}

func TestMissingConfigIsNotReported(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "app")
	_, _, err := Spec{Config: missing}.Load(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorIs(t, err, report.ErrDontReport)
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{
		"-c", "user.email=test@example.com",
		"-c", "user.name=test",
		"-c", "commit.gpgsign=false",
	}, args...)
	cmd := execdriver.MustCommand(t.Context(), "git", full...)
	cmd.Dir = dir
	require.NoError(t, execdriver.Run(t.Context(), cmd))
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
