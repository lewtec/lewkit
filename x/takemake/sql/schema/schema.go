package schema

import "github.com/lewtec/lewkit/x/takemake/sql"

// Database is the catalog after migrations run.
type Database struct {
	// Dialect is the dialect of the logs applied so far.
	Dialect sql.Dialect
	// Tables is the catalog in creation order.
	Tables []*Table

	byName  map[string]*Table
	indexes map[string]string
}

// Table is one table in the catalog.
type Table struct {
	// Name is the table name.
	Name string
	// Columns is the column list in order.
	Columns []*Column
	// Primary is the primary key in order.
	Primary []string
	// Uniques is table-level UNIQUE constraints.
	Uniques [][]string
	// Foreign is the foreign keys.
	Foreign []*ForeignKey
	// Indexes are the indexes on this table.
	Indexes []*Index
	// Checks is the CHECK expressions.
	Checks []sql.Expr
}

// Column is one column in a catalog table.
type Column struct {
	// Name is the column name.
	Name string
	// Type is the column type.
	Type sql.Type
	// Primary means the column is part of the primary key.
	Primary bool
	// Auto means AUTOINCREMENT.
	Auto bool
	// NotNull means NOT NULL.
	NotNull bool
	// IsUnique means UNIQUE on this column.
	IsUnique bool
	// Default is the DEFAULT expression.
	Default *sql.Expr
}

// ForeignKey is a foreign key in the catalog.
type ForeignKey struct {
	// Columns are the local columns.
	Columns []string
	// Table is the referenced table.
	Table string
	// RefColumns are the referenced columns.
	RefColumns []string
	// OnDelete is the ON DELETE action.
	OnDelete sql.Action
	// OnUpdate is the ON UPDATE action.
	OnUpdate sql.Action
}

// Index is one index in the catalog.
type Index struct {
	// Name is the index name.
	Name string
	// IsUnique means the index is unique.
	IsUnique bool
	// Columns are the indexed columns.
	Columns []string
	// Where is a partial-index predicate.
	Where *sql.Expr
}

// New returns an empty catalog.
func New() *Database {
	return &Database{
		byName:  map[string]*Table{},
		indexes: map[string]string{},
	}
}

func (db *Database) init() {
	if db.byName == nil {
		db.byName = map[string]*Table{}
	}
	if db.indexes == nil {
		db.indexes = map[string]string{}
	}
}

// Table returns the named table, or nil when the catalog has no such table.
func (db *Database) Table(name string) *Table {
	if db == nil {
		return nil
	}
	return db.byName[name]
}

// Column returns the named column, or nil when the table has no such column.
func (t *Table) Column(name string) *Column {
	if t == nil {
		return nil
	}
	for _, col := range t.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

// ForeignKey returns the first foreign key that references table.
func (t *Table) ForeignKey(table string) *ForeignKey {
	if t == nil {
		return nil
	}
	for _, fk := range t.Foreign {
		if fk.Table == table {
			return fk
		}
	}
	return nil
}

// Index returns the named index, or nil when the table has no such index.
func (t *Table) Index(name string) *Index {
	if t == nil {
		return nil
	}
	for _, idx := range t.Indexes {
		if idx.Name == name {
			return idx
		}
	}
	return nil
}
