package host

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/report"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"github.com/lewtec/lewkit/x/http/site"
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

func TestSpecEntryFlagBeatsFileAndMain(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other")
	missing := filepath.Join(dir, "missing")
	require.NoError(t, os.Mkdir(other, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "static:missing.zip",
		"go_main": "."
	}`), 0o644))

	cfg, _, err := Spec{
		Dir:    dir,
		Config: filepath.Join(dir, "eletrocromo.json"),
		Entry:  "go:other",
		GoMain: missing,
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, other, cfg.GoMain)
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

func TestStaticExamplesLoad(t *testing.T) {
	root := moduleRoot(t)

	t.Run("site", func(t *testing.T) {
		dir := filepath.Join(root, "examples", "app", "site")
		cfg, base, err := Spec{Config: dir}.Load(t.Context())
		require.NoError(t, err)
		require.Equal(t, dir, base)
		require.Equal(t, "br.tec.lew.eletrocromo.site", cfg.PackageID)
		require.Equal(t, "Site", cfg.AppName)
		t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
		body, err := os.ReadFile(filepath.Join(cfg.GoMain, "main.go"))
		require.NoError(t, err)
		require.Contains(t, string(body), "site.Dir")
		require.Contains(t, string(body), `"Site"`)
		page, err := os.ReadFile(filepath.Join(cfg.GoMain, "site", "about.html"))
		require.NoError(t, err)
		require.Contains(t, string(page), "<h1>about</h1>")
	})

	t.Run("zip", func(t *testing.T) {
		dir := filepath.Join(root, "examples", "app", "zip")
		cfg, base, err := Spec{Config: dir}.Load(t.Context())
		require.NoError(t, err)
		require.Equal(t, dir, base)
		require.Equal(t, "br.tec.lew.eletrocromo.zip", cfg.PackageID)
		t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
		body, err := os.ReadFile(filepath.Join(cfg.GoMain, "main.go"))
		require.NoError(t, err)
		require.Contains(t, string(body), "site.Zip")
		_, err = os.Stat(filepath.Join(cfg.GoMain, "site.zip"))
		require.NoError(t, err)

		raw, err := os.ReadFile(filepath.Join(dir, "site.zip"))
		require.NoError(t, err)
		handler, err := site.Zip(t.Context(), bytes.NewReader(raw))
		require.NoError(t, err)
		home := httptest.NewRequest(http.MethodGet, "/", nil)
		homeResponse := httptest.NewRecorder()
		handler.ServeHTTP(homeResponse, home)
		require.Equal(t, http.StatusOK, homeResponse.Code)
		require.Contains(t, homeResponse.Body.String(), "<h1>zip</h1>")
		note := httptest.NewRequest(http.MethodGet, "/note.txt", nil)
		note.Header.Set("Range", "bytes=0-4")
		noteResponse := httptest.NewRecorder()
		handler.ServeHTTP(noteResponse, note)
		require.Equal(t, http.StatusPartialContent, noteResponse.Code)
		require.Equal(t, "range", noteResponse.Body.String())
	})
}

func TestExampleConfigsSetEntry(t *testing.T) {
	root := moduleRoot(t)
	var count int
	err := filepath.WalkDir(filepath.Join(root, "examples"), func(path string, item os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.IsDir() {
			switch item.Name() {
			case "node_modules", "dist", ".git":
				return filepath.SkipDir
			}
			return nil
		}
		if item.Name() != "eletrocromo.json" {
			return nil
		}
		raw, err := os.ReadFile(path)
		require.NoError(t, err)
		var doc struct {
			Entry       string `json:"entry"`
			GoMain      string `json:"go_main"`
			VersionName string `json:"version_name"`
		}
		require.NoError(t, json.Unmarshal(raw, &doc))
		require.NotEmpty(t, doc.Entry, path)
		require.Empty(t, doc.GoMain, path)
		require.Empty(t, doc.VersionName, path)
		count++
		return nil
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, count, 30)
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
