package db_test

import (
	"testing"

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
