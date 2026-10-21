package entry_test

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/entry"
	_ "github.com/lewtec/lewkit/x/build/entry/prelude"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	_ "github.com/lewtec/lewkit/x/driver/exec/prelude"
	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestParseEntry(t *testing.T) {
	got, err := entry.ParseEntry("", "")
	require.NoError(t, err)
	require.Equal(t, entry.Entry{Name: "go", Ref: "."}, got)

	got, err = entry.ParseEntry("", "./cmd/app")
	require.NoError(t, err)
	require.Equal(t, entry.Entry{Name: "go", Ref: "./cmd/app"}, got)

	got, err = entry.ParseEntry("go:./cmd/main", "ignored")
	require.NoError(t, err)
	require.Equal(t, entry.Entry{Name: "go", Ref: "./cmd/main"}, got)

	got, err = entry.ParseEntry("astro:.", "")
	require.NoError(t, err)
	require.Equal(t, entry.Entry{Name: "astro", Ref: "."}, got)

	got, err = entry.ParseEntry("static:site.zip", "")
	require.NoError(t, err)
	require.Equal(t, entry.Entry{Name: "static", Ref: "site.zip"}, got)

	_, err = entry.ParseEntry("nope", "")
	require.ErrorIs(t, err, entry.ErrEntrySyntax)

	_, err = entry.ParseEntry("static:", "")
	require.ErrorIs(t, err, entry.ErrEntrySyntax)

	_, err = entry.ParseEntry("nope:x", "")
	require.ErrorIs(t, err, entry.ErrEntryProvider)
	require.ErrorIs(t, err, errs.ErrUnknown)
}

func TestMaterializeStatic(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "site.zip")
	require.NoError(t, os.WriteFile(zipPath, zipWithIndex(t), 0o644))

	first, err := entry.Main(t.Context(), "static:site.zip", "", dir, "Milhao")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(first) })
	second, err := entry.Main(t.Context(), "static:site.zip", "", dir, "Milhao")
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
	dirMain, err := entry.Main(t.Context(), "static:web", "ignored", dir, "")
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
	main, err := entry.Main(t.Context(), "static:site.zip", "", dir, "Site")
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
	_, err := entry.Main(t.Context(), "static:missing.zip", "", dir, "app")
	require.ErrorIs(t, err, entry.ErrStaticMissing)

	_, err = entry.Main(t.Context(), "astro:.", "", dir, "app")
	require.ErrorIs(t, err, entry.ErrAstroConfig)
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
