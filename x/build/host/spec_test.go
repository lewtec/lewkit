package host

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/report"
	"github.com/lewtec/lewkit/x/build/entry"
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

func TestSpecFlagsOnlyStaticZip(t *testing.T) {
	dir := t.TempDir()
	writeSiteZip(t, filepath.Join(dir, "city-racing.zip"), "<h1>city</h1>")

	cfg, base, err := Spec{
		Dir:   dir,
		Entry: "static:city-racing.zip",
		ID:    "  teste.cityracing  ",
	}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, dir, base)
	require.Equal(t, "teste.cityracing", cfg.PackageID)
	require.Equal(t, "city-racing", cfg.AppName)
	body, err := os.ReadFile(filepath.Join(cfg.GoMain, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(body), `"city-racing"`)
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>city</h1>")
}

func TestSpecFlagZipPrefersWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	writeSiteZip(t, filepath.Join(configDir, "city-racing.zip"), "<h1>config</h1>")
	writeSiteZip(t, filepath.Join(workDir, "city-racing.zip"), "<h1>cwd</h1>")
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "static:old.zip"
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{
		Dir:    configDir,
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "static:city-racing.zip",
		ID:     "teste.cityracing",
	}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, "teste.cityracing", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>cwd</h1>")
}

func TestSpecFlagZipFallsBackToConfigDir(t *testing.T) {
	configDir := t.TempDir()
	writeSiteZip(t, filepath.Join(configDir, "city-racing.zip"), "<h1>config</h1>")
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(t.TempDir())

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "static:city-racing.zip",
	}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, "br.tec.lew.file", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>config</h1>")
}

func TestSpecFileEntryIgnoresWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	writeSiteZip(t, filepath.Join(configDir, "city-racing.zip"), "<h1>config</h1>")
	writeSiteZip(t, filepath.Join(workDir, "city-racing.zip"), "<h1>cwd</h1>")
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "static:city-racing.zip"
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{Config: filepath.Join(configDir, "eletrocromo.json")}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	copied, err := os.ReadFile(filepath.Join(cfg.GoMain, "site.zip"))
	require.NoError(t, err)
	require.Contains(t, string(copied), "<h1>config</h1>")
	require.NotContains(t, string(copied), "<h1>cwd</h1>")
}

func TestSpecStaticDotStaysConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "index.html"), []byte("from-config"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "index.html"), []byte("from-cwd"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "static:.",
	}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	body, err := os.ReadFile(filepath.Join(cfg.GoMain, "site", "index.html"))
	require.NoError(t, err)
	require.Equal(t, "from-config", string(body))
	require.Equal(t, "File", cfg.AppName)
}

func TestSpecAstroDotStaysConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(workDir)

	_, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "astro:.",
	}.Load(t.Context())
	require.ErrorIs(t, err, entry.ErrAstroConfig)
	require.Contains(t, err.Error(), configDir)
	require.NotContains(t, err.Error(), workDir)
}

func TestSpecGoFlagStaysBesideConfig(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(configDir, "other"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "other"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "static:missing.zip"
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "go:other",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(configDir, "other"), cfg.GoMain)
}

func TestSpecMainFlagPrefersWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(configDir, "cmd"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(workDir, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		GoMain: "cmd",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(workDir, "cmd"), cfg.GoMain)
}

func TestSpecMainFlagFallsBackToConfigDir(t *testing.T) {
	configDir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(configDir, "cmd"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(t.TempDir())

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		GoMain: "cmd",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(configDir, "cmd"), cfg.GoMain)
}

func TestSpecNameFlagBeatsAssetStem(t *testing.T) {
	dir := t.TempDir()
	writeSiteZip(t, filepath.Join(dir, "city-racing.zip"), "<h1>city</h1>")

	cfg, _, err := Spec{
		Dir:   dir,
		Entry: "static:city-racing.zip",
		ID:    "teste.cityracing",
		Name:  "City Racing",
	}.Load(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(cfg.GoMain) })
	require.Equal(t, "City Racing", cfg.AppName)
	body, err := os.ReadFile(filepath.Join(cfg.GoMain, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(body), `"City Racing"`)
}

func TestSpecNameFromPackageID(t *testing.T) {
	dir := t.TempDir()
	cfg, _, err := Spec{Dir: dir, ID: "teste.cityracing", Entry: "go:."}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "cityracing", cfg.AppName)
	require.Equal(t, dir, cfg.GoMain)
}

func TestSpecTrimsFlags(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))

	cfg, _, err := Spec{
		Dir:     dir,
		Config:  filepath.Join(dir, "eletrocromo.json"),
		ID:      "  teste.cityracing ",
		Name:    "  City Racing  ",
		Version: "  1.2.3  ",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "teste.cityracing", cfg.PackageID)
	require.Equal(t, "City Racing", cfg.AppName)
	require.Equal(t, "1.2.3", cfg.VersionName)
}

func TestSpecBlankFlagsKeepFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))

	cfg, _, err := Spec{
		Dir:    dir,
		Config: filepath.Join(dir, "eletrocromo.json"),
		ID:     "   ",
		Name:   "\n",
		Entry:  "  ",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "br.tec.lew.file", cfg.PackageID)
	require.Equal(t, "File", cfg.AppName)
	require.Equal(t, dir, cfg.GoMain)
}

func TestSpecIconFlagPrefersWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "mark.png"), []byte("config"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(workDir, "mark.png"), []byte("cwd"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:.",
		"icon": "mark.png"
	}`), 0o644))
	t.Chdir(workDir)

	cfg, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Icon:   "mark.png",
	}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, filepath.Join(workDir, "mark.png"), cfg.Icon)

	kept, _, err := Spec{Config: filepath.Join(configDir, "eletrocromo.json")}.Load(t.Context())
	require.NoError(t, err)
	require.Equal(t, "mark.png", kept.Icon)
}

func TestSpecMissingFlagZipNamesWorkingDirectory(t *testing.T) {
	workDir := t.TempDir()
	configDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "eletrocromo.json"), []byte(`{
		"package_id": "br.tec.lew.file",
		"app_name": "File",
		"entry": "go:."
	}`), 0o644))
	t.Chdir(workDir)

	_, _, err := Spec{
		Config: filepath.Join(configDir, "eletrocromo.json"),
		Entry:  "static:city-racing.zip",
		ID:     "teste.cityracing",
	}.Load(t.Context())
	require.ErrorIs(t, err, entry.ErrStaticMissing)
	require.Contains(t, err.Error(), filepath.Join(workDir, "city-racing.zip"))
}

func TestSpecExplicitMissingConfigIgnoresFlags(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "gone.json")
	_, _, err := Spec{
		Config: missing,
		Entry:  "static:city-racing.zip",
		ID:     "teste.cityracing",
	}.Load(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorIs(t, err, report.ErrDontReport)
	require.Contains(t, err.Error(), missing)
}

func TestMissingConfigIsNotReported(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "app")
	_, _, err := Spec{Config: missing}.Load(t.Context())
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorIs(t, err, report.ErrDontReport)
}

func writeSiteZip(t *testing.T, path, body string) {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	header := &zip.FileHeader{Name: "index.html", Method: zip.Store}
	file, err := writer.CreateHeader(header)
	require.NoError(t, err)
	_, err = file.Write([]byte(body))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
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
