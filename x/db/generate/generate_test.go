package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCompare(t *testing.T) {
	dir := filepath.Join("testdata", "ok")
	engines, err := scan(dir)
	require.NoError(t, err)
	require.Len(t, engines, 2)
	require.NoError(t, compareQueries(engines))
	assert.Equal(t, "one", engines[0].named["GetItem"])
}

func TestScanMismatch(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sqlite", "migrations"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "postgres", "migrations"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sqlite", "q.sql"), []byte("-- name: A :one\nSELECT 1;\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "postgres", "q.sql"), []byte("-- name: B :one\nSELECT 1;\n"), 0o644))
	engines, err := scan(dir)
	require.NoError(t, err)
	err = compareQueries(engines)
	require.Error(t, err)
}

func TestWriteSQLC(t *testing.T) {
	dir := t.TempDir()
	engines := []engine{{
		dir:   "sqlite",
		sqlc:  "sqlite",
		files: []string{"sqlite/items.sql"},
	}}
	require.NoError(t, writeSQLC(dir, engines))
	b, err := os.ReadFile(filepath.Join(dir, "sqlc.yaml"))
	require.NoError(t, err)
	assert.Contains(t, string(b), "engine: sqlite")
	assert.Contains(t, string(b), "emit_interface: true")
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join("testdata", "ok")
	require.NoError(t, copyDir(src, dir))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/store\n\ngo 1.27\n"), 0o644))
	require.NoError(t, Run(t.Context(), dir))
	b, err := os.ReadFile(filepath.Join(dir, "zz_queries.go"))
	require.NoError(t, err)
	got := string(b)
	assert.Contains(t, got, "type Queries interface")
	assert.Contains(t, got, "ListItems")
	assert.Contains(t, got, "func New")
	assert.Contains(t, got, "func Open")
	fsb, err := os.ReadFile(filepath.Join(dir, "zz_fs.go"))
	require.NoError(t, err)
	assert.Contains(t, string(fsb), "//go:embed")
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}
