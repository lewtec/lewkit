package db_test

import (
	"database/sql"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db"
)

func TestParseDatabaseFlag(t *testing.T) {
	type args struct {
		DB db.Arg[struct{}] `long:"database"`
	}
	got, err := cmd.Parse[args]("--database", "sqlite://file.db")
	require.NoError(t, err)
	require.NotNil(t, got.DB.Value())
	assert.Equal(t, "sqlite://file.db", got.DB.Value().URL())
}

func TestFromURL(t *testing.T) {
	a, err := db.FromURL[struct{}]("sqlite://file.db")
	require.NoError(t, err)
	require.NotNil(t, a.Value())
	assert.Equal(t, "sqlite://file.db", a.Value().URL())
}

func TestParseRequired(t *testing.T) {
	type args struct {
		DB db.Arg[struct{}] `long:"database"`
	}
	_, err := cmd.Parse[args]()
	assert.ErrorIs(t, err, cmd.ErrMissingValue)
}

func TestScheme(t *testing.T) {
	assert.Equal(t, "sqlite", db.Scheme(":memory:"))
	assert.Equal(t, "sqlite", db.Scheme("file.db"))
	assert.Equal(t, "sqlite", db.Scheme("sqlite:///tmp/x.db"))
	assert.Equal(t, "sqlite", db.Scheme("file:foo.db"))
	assert.Equal(t, "postgres", db.Scheme("postgres://localhost/app"))
	assert.Equal(t, "postgres", db.Scheme("postgresql://localhost/app"))
}

func TestOneRootPerArg(t *testing.T) {
	type args struct {
		Primary db.Arg[struct{}] `long:"primary"`
		Cache   db.Arg[struct{}] `long:"cache"`
	}
	nop := db.Connector{Driver: "unused", Up: func(*sql.DB, fs.FS) error { return nil }}
	db.Register("sqlite", nop)
	db.Register("postgres", nop)
	got, err := cmd.Parse[args]("--primary", "sqlite://a.db", "--cache", "postgres://localhost/b")
	require.NoError(t, err)
	primary := fstest.MapFS{"sqlite/migrations/1.up.sql": {Data: []byte("--")}}
	cache := fstest.MapFS{"postgres/migrations/1.up.sql": {Data: []byte("--")}}
	err = got.Primary.Migrate(t.Context(), cache)
	require.ErrorIs(t, err, db.ErrNoMigrations)
	assert.Contains(t, err.Error(), "sqlite/migrations")
	err = got.Cache.Migrate(t.Context(), primary)
	require.ErrorIs(t, err, db.ErrNoMigrations)
	assert.Contains(t, err.Error(), "postgres/migrations")
}

func TestNoMigrationsForEngine(t *testing.T) {
	db.Register("postgres", db.Connector{
		Driver: "unused",
		Up:     func(*sql.DB, fs.FS) error { return nil },
	})
	root := fstest.MapFS{
		"sqlite/migrations/000001_items.up.sql": {Data: []byte("select 1;")},
	}
	var a db.Arg[struct{}]
	require.NoError(t, a.Parse("postgres://localhost/app"))
	err := a.Value().Migrate(t.Context(), root)
	require.Error(t, err)
	assert.ErrorIs(t, err, db.ErrNoMigrations)
	assert.Contains(t, err.Error(), "postgres/migrations")
}

func TestUnknownScheme(t *testing.T) {
	var a db.Arg[struct{}]
	require.NoError(t, a.Parse("mysql://localhost/db"))
	err := a.Value().Migrate(t.Context(), nil)
	assert.ErrorIs(t, err, db.ErrUnknownScheme)
}

func TestConnCloseNil(t *testing.T) {
	var c *db.Conn[struct{}]
	assert.NoError(t, c.Close())
	assert.Equal(t, "", c.URL())
}
