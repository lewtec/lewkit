package sqlite_test

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db"
	"github.com/lewtec/lewkit/x/db/sqlite"

	_ "modernc.org/sqlite"
)

//go:embed testdata/migrations/*.sql
var migrationsFS embed.FS

type mig struct{ sqlite.Engine }

func (mig) FS() fs.FS   { return migrationsFS }
func (mig) Dir() string { return "testdata/migrations" }

type queries struct {
	db *sql.DB
}

func newQueries(conn *sql.DB) *queries {
	return &queries{db: conn}
}

func (q *queries) Count(ctx context.Context) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, "select count(*) from items").Scan(&n)
	return n, err
}

func TestOpenStdlibMigratesAndQueries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	s := db.FromURL[*queries, mig](path)
	h, err := s.OpenStdlib(t.Context(), newQueries)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, h.Close()) })

	n, err := h.Q.Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	h2, err := s.OpenStdlib(t.Context(), newQueries)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, h2.Close()) })
	n, err = h2.Q.Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestOpenStdlibMemory(t *testing.T) {
	s := db.FromURL[*queries, mig](":memory:")
	h, err := s.OpenStdlib(t.Context(), newQueries)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, h.Close()) })
	n, err := h.Q.Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestParseThenOpen(t *testing.T) {
	type args struct {
		DB db.SQL[*queries, mig] `long:"database"`
	}
	path := filepath.Join(t.TempDir(), "parsed.db")
	got, err := cmd.Parse[args]("--database", path)
	require.NoError(t, err)
	h, err := got.DB.OpenStdlib(t.Context(), newQueries)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, h.Close()) })
	n, err := h.Q.Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}
