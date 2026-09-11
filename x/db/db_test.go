package db_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/db"
)

func init() {
	sql.Register("dbtest", stubDriver{})
}

type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) { return stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (stubConn) Close() error                        { return nil }
func (stubConn) Begin() (driver.Tx, error)           { return nil, errors.New("unused") }
func (stubConn) Ping(context.Context) error          { return nil }

type src struct{}

func (src) FS() fs.FS       { return fstest.MapFS{} }
func (src) Dir() string     { return "." }
func (src) Driver() string  { return "dbtest" }
func (src) Dialect() string { return "missing" }

type twoEngines struct {
	PG   db.SQL[struct{}, src] `long:"postgres" help:"postgres URL"`
	Lite db.SQL[struct{}, src] `long:"sqlite" help:"sqlite path"`
}

func TestParseTwoEngines(t *testing.T) {
	got, err := cmd.Parse[twoEngines]("--postgres", "postgres://db", "--sqlite", "file.db")
	require.NoError(t, err)
	assert.Equal(t, "postgres://db", got.PG.URL.Value())
	assert.Equal(t, "file.db", got.Lite.URL.Value())
}

func TestParseSingleFlag(t *testing.T) {
	type one struct {
		DB db.SQL[struct{}, src] `long:"database" help:"database URL"`
	}
	got, err := cmd.Parse[one]("--database", "postgres://db")
	require.NoError(t, err)
	assert.Equal(t, "postgres://db", got.DB.URL.Value())
}

func TestFromURL(t *testing.T) {
	s := db.FromURL[struct{}, src]("file.db")
	assert.Equal(t, "file.db", s.URL.Value())
}

func TestMigrateUnknownDialect(t *testing.T) {
	s := db.FromURL[struct{}, src]("file:unused.db")
	err := s.Migrate(t.Context())
	require.Error(t, err)
	assert.ErrorIs(t, err, db.ErrUnknownDialect)
}

func TestOpenMemoryRejected(t *testing.T) {
	s := db.FromURL[struct{}, src](":memory:")
	err := s.Migrate(t.Context())
	require.ErrorIs(t, err, db.ErrEphemeral)
}

func TestHandleCloseNil(t *testing.T) {
	var h *db.Handle[struct{}]
	assert.NoError(t, h.Close())
}
