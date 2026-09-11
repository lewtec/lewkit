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
	_ "github.com/lewtec/lewkit/x/db/sqlite"
)

//go:embed testdata/migrations/*.sql
var migrationsFS embed.FS

func migrations(t *testing.T) fs.FS {
	t.Helper()
	return db.Dir(migrationsFS, "testdata/migrations")
}

type querier interface {
	Count(context.Context) (int, error)
}

type queries struct {
	db db.DBTX
}

func newQuerier(conn db.DBTX) querier {
	return &queries{db: conn}
}

func (q *queries) Count(ctx context.Context) (int, error) {
	var n int
	err := q.db.QueryRowContext(ctx, "select count(*) from items").Scan(&n)
	return n, err
}

func TestArgValueOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.db")
	type args struct {
		DB db.Arg[querier] `long:"database"`
	}
	got, err := cmd.Parse[args]("--database", "sqlite://"+path)
	require.NoError(t, err)
	d := got.DB.Value()
	require.NotNil(t, d)
	require.NoError(t, d.Open(t.Context(), migrations(t), newQuerier))
	t.Cleanup(func() { require.NoError(t, d.Close()) })

	n, err := d.Queries().Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	err = d.Tx(t.Context(), func(q querier) error {
		n, err := q.Count(t.Context())
		if err != nil {
			return err
		}
		if n != 1 {
			t.Fatalf("tx count %d", n)
		}
		return nil
	})
	require.NoError(t, err)
}

func TestMemoryAndBarePath(t *testing.T) {
	var a db.Arg[querier]
	require.NoError(t, a.Parse(":memory:"))
	d := a.Value()
	require.NoError(t, d.Open(t.Context(), migrations(t), newQuerier))
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	n, err := d.Queries().Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	path := filepath.Join(t.TempDir(), "bare.db")
	require.NoError(t, a.Parse(path))
	d = a.Value()
	require.NoError(t, d.Open(t.Context(), migrations(t), newQuerier))
	t.Cleanup(func() { require.NoError(t, d.Close()) })
	n, err = d.Queries().Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

func TestMigrateIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "twice.db")
	var a db.Arg[querier]
	require.NoError(t, a.Parse("sqlite://"+path))
	require.NoError(t, a.Value().Open(t.Context(), migrations(t), newQuerier))
	require.NoError(t, a.Value().Close())
	require.NoError(t, a.Value().Open(t.Context(), migrations(t), newQuerier))
	t.Cleanup(func() { require.NoError(t, a.Value().Close()) })
	n, err := a.Value().Queries().Count(t.Context())
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}

var _ querier = (*queries)(nil)
var _ db.DBTX = (*sql.DB)(nil)
var _ cmd.Valuer[*db.Conn[querier]] = db.Arg[querier]{}
