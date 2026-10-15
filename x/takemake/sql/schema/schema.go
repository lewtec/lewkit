package schema

import (
	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/takemake/sql"
)

// Database is the catalog after migrations run.
type Database struct{}

// Table is one table in the catalog.
type Table struct {
	// Name is the table name.
	Name string
}

// Table returns the named table. The scaffold keeps the name only.
func (db *Database) Table(name string) *Table {
	if db == nil {
		return nil
	}
	return &Table{Name: name}
}

// New returns an empty catalog.
func New() *Database {
	return &Database{}
}

// Apply folds file onto the catalog, in order.
func (db *Database) Apply(file *sql.File) error {
	if db == nil || file == nil {
		return errs.ErrScaffold
	}
	return errs.ErrScaffold
}

// Make returns the CreateTable snapshot of the catalog.
func (db *Database) Make() (*sql.File, error) {
	if db == nil {
		return nil, errs.ErrScaffold
	}
	return nil, errs.ErrScaffold
}
