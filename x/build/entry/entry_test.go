package entry

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestParseEntry(t *testing.T) {
	got, err := ParseEntry("", "")
	require.NoError(t, err)
	require.Equal(t, Entry{Provider: ProviderGo, Ref: "."}, got)

	got, err = ParseEntry("", "./cmd/app")
	require.NoError(t, err)
	require.Equal(t, Entry{Provider: ProviderGo, Ref: "./cmd/app"}, got)

	got, err = ParseEntry("go:./cmd/main", "ignored")
	require.NoError(t, err)
	require.Equal(t, Entry{Provider: ProviderGo, Ref: "./cmd/main"}, got)

	got, err = ParseEntry("astro:.", "")
	require.NoError(t, err)
	require.Equal(t, Entry{Provider: ProviderAstro, Ref: "."}, got)

	got, err = ParseEntry("static:site.zip", "")
	require.NoError(t, err)
	require.Equal(t, Entry{Provider: ProviderStatic, Ref: "site.zip"}, got)

	_, err = ParseEntry("nope", "")
	require.ErrorIs(t, err, ErrEntrySyntax)

	_, err = ParseEntry("static:", "")
	require.ErrorIs(t, err, ErrEntrySyntax)

	_, err = ParseEntry("nope:x", "")
	require.ErrorIs(t, err, ErrEntryProvider)
	require.ErrorIs(t, err, errs.ErrUnknown)
}

func TestMaterializeStatic(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "site.zip")
	require.NoError(t, os.WriteFile(zipPath, zipWithIndex(t), 0o644))

	first, err := Main(t.Context(), "static:site.zip", "", dir, "Milhao")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(first) })
	second, err := Main(t.Context(), "static:site.zip", "", dir, "Milhao")
	require.NoError(t, err)
	require.Equal(t, first, second)

	body, err := os.ReadFile(filepath.Join(first, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(body), "site.Zip")
	require.Contains(t, string(body), `"Milhao"`)
	_, err = os.Stat(filepath.Join(first, "site.zip"))
	require.NoError(t, err)

	web := filepath.Join(dir, "web")
	require.NoError(t, os.Mkdir(web, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(web, "index.html"), []byte("hi"), 0o644))
	dirMain, err := Main(t.Context(), "static:web", "ignored", dir, "")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dirMain) })
	body, err = os.ReadFile(filepath.Join(dirMain, "main.go"))
	require.NoError(t, err)
	require.Contains(t, string(body), "site.Dir")
	require.Contains(t, string(body), `"app"`)
	copied, err := os.ReadFile(filepath.Join(dirMain, "site", "index.html"))
	require.NoError(t, err)
	require.Equal(t, "hi", string(copied))
}

func TestStaticModuleBuilds(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "site.zip"), zipWithIndex(t), 0o644))
	main, err := Main(t.Context(), "static:site.zip", "", dir, "Site")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(main) })

	out := filepath.Join(t.TempDir(), "app")
	cmd := execdriver.MustCommand(t.Context(), "go", "build", "-o", out, ".")
	cmd.Dir = main
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	require.NoError(t, execdriver.Run(t.Context(), cmd))
	info, err := os.Stat(out)
	require.NoError(t, err)
	require.Greater(t, info.Size(), int64(0))
}

func TestMainReportsMissingSites(t *testing.T) {
	dir := t.TempDir()
	_, err := Main(t.Context(), "static:missing.zip", "", dir, "app")
	require.ErrorIs(t, err, ErrStaticMissing)

	_, err = Main(t.Context(), "astro:.", "", dir, "app")
	require.ErrorIs(t, err, ErrAstroConfig)
}

func TestAssembleAstro(t *testing.T) {
	t.Run("server", func(t *testing.T) {
		project := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "server"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "server", "entry.mjs"), []byte("export default {}\n"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "client"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "client", "app.js"), []byte("x"), 0o644))
		worker := filepath.Join(t.TempDir(), "worker")
		require.NoError(t, assembleAstro(project, worker))
		_, err := os.Stat(filepath.Join(worker, "entry.mjs"))
		require.NoError(t, err)
		body, err := os.ReadFile(filepath.Join(worker, "assets", "app.js"))
		require.NoError(t, err)
		require.Equal(t, "x", string(body))
	})

	t.Run("legacy", func(t *testing.T) {
		project := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(project, "dist", "_worker.js"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "_worker.js", "index.js"), []byte("export default {}\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(project, "dist", "app.css"), []byte("body{}\n"), 0o644))
		worker := filepath.Join(t.TempDir(), "worker")
		require.NoError(t, assembleAstro(project, worker))
		_, err := os.Stat(filepath.Join(worker, "entry.mjs"))
		require.NoError(t, err)
		body, err := os.ReadFile(filepath.Join(worker, "assets", "app.css"))
		require.NoError(t, err)
		require.Equal(t, "body{}\n", string(body))
	})
}

func TestAstroLock(t *testing.T) {
	dir := t.TempDir()
	name, args, ok := astroLock(dir)
	require.False(t, ok)
	require.Empty(t, name)
	require.Nil(t, args)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "bun.lock"), []byte("{}\n"), 0o644))
	name, args, ok = astroLock(dir)
	require.True(t, ok)
	require.Equal(t, "bun", name)
	require.Equal(t, []string{"run", "build"}, args)

	yarn := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(yarn, "yarn.lock"), []byte("# yarn\n"), 0o644))
	name, args, ok = astroLock(yarn)
	require.True(t, ok)
	require.Equal(t, "yarn", name)
	require.Equal(t, []string{"build"}, args)
}

func zipWithIndex(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	file, err := writer.Create("index.html")
	require.NoError(t, err)
	_, err = file.Write([]byte("<h1>home</h1>"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes()
}
