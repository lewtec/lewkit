package schema

import (
	"fmt"

	"github.com/lewtec/lewkit/x/takemake/sql"
)

// Apply folds file onto the catalog, in order.
// Select, Insert, Update, and Delete do not change tables.
func (db *Database) Apply(file *sql.File) error {
	if db == nil || file == nil {
		return fmt.Errorf("%w: catalog", ErrInvalid)
	}
	db.init()
	if err := db.useDialect(file.Dialect); err != nil {
		return err
	}
	for _, stmt := range file.Statements {
		if err := db.apply(stmt); err != nil {
			return err
		}
	}
	return nil
}

func (db *Database) useDialect(dialect sql.Dialect) error {
	if !dialect.IsValid() {
		dialect = sql.SQLite
	}
	if !db.Dialect.IsValid() {
		db.Dialect = dialect
		return nil
	}
	if db.Dialect != dialect {
		return fmt.Errorf("%w: dialect %s", ErrInvalid, dialect)
	}
	return nil
}

func (db *Database) apply(stmt sql.Statement) error {
	switch stmt := stmt.(type) {
	case nil:
		return fmt.Errorf("%w: statement", ErrInvalid)
	case *sql.CreateTable:
		return db.applyCreate(stmt)
	case *sql.AlterTable:
		return db.applyAlter(stmt)
	case *sql.DropTable:
		return db.applyDrop(stmt)
	case *sql.CreateIndex:
		return db.applyIndex(stmt)
	case *sql.Select, *sql.Insert, *sql.Update, *sql.Delete:
		return nil
	default:
		return fmt.Errorf("%w: statement", ErrInvalid)
	}
}

func (db *Database) applyCreate(stmt *sql.CreateTable) error {
	if stmt.Name == "" {
		return fmt.Errorf("%w: table name", ErrInvalid)
	}
	if _, ok := db.byName[stmt.Name]; ok {
		if stmt.UnlessExists {
			return nil
		}
		return fmt.Errorf("%w: table %s", ErrDuplicate, stmt.Name)
	}
	table := &Table{Name: stmt.Name}
	seen := map[string]bool{}
	var inline []string
	for _, col := range stmt.Columns {
		if col == nil || col.Name == "" || !col.Type.IsValid() {
			return fmt.Errorf("%w: column", ErrInvalid)
		}
		if seen[col.Name] {
			return fmt.Errorf("%w: column %s", ErrDuplicate, col.Name)
		}
		seen[col.Name] = true
		table.Columns = append(table.Columns, &Column{
			Name:     col.Name,
			Type:     col.Type,
			Primary:  col.Primary,
			Auto:     col.Auto,
			NotNull:  col.Required,
			IsUnique: col.IsUnique,
			Default:  cloneExpr(col.DefaultExpr),
		})
		if col.Primary {
			inline = append(inline, col.Name)
		}
		if col.RefTable != "" {
			table.Foreign = append(table.Foreign, &ForeignKey{
				Columns:    []string{col.Name},
				Table:      col.RefTable,
				RefColumns: copyStrings(col.RefColumns),
				OnDelete:   col.DeleteAction,
				OnUpdate:   col.UpdateAction,
			})
		}
	}
	if len(stmt.Key) > 0 && len(inline) > 0 {
		return fmt.Errorf("%w: primary key", ErrInvalid)
	}
	pk := copyStrings(stmt.Key)
	if len(pk) == 0 {
		pk = inline
	}
	for _, name := range pk {
		col := table.Column(name)
		if col == nil {
			return fmt.Errorf("%w: primary key %s", ErrInvalid, name)
		}
		col.Primary = true
	}
	table.Primary = pk
	for _, cols := range stmt.Uniques {
		for _, name := range cols {
			if table.Column(name) == nil {
				return fmt.Errorf("%w: unique %s", ErrInvalid, name)
			}
		}
		table.Uniques = append(table.Uniques, copyStrings(cols))
	}
	for _, fk := range stmt.Foreign {
		if fk == nil || fk.RefTable == "" || len(fk.Columns) == 0 {
			return fmt.Errorf("%w: foreign key", ErrInvalid)
		}
		for _, name := range fk.Columns {
			if table.Column(name) == nil {
				return fmt.Errorf("%w: foreign key %s", ErrInvalid, name)
			}
		}
		table.Foreign = append(table.Foreign, &ForeignKey{
			Columns:    copyStrings(fk.Columns),
			Table:      fk.RefTable,
			RefColumns: copyStrings(fk.RefColumns),
			OnDelete:   fk.DeleteAction,
			OnUpdate:   fk.UpdateAction,
		})
	}
	for _, check := range stmt.Checks {
		table.Checks = append(table.Checks, cloneExprValue(check))
	}
	db.Tables = append(db.Tables, table)
	db.byName[table.Name] = table
	return nil
}

func (db *Database) applyAlter(stmt *sql.AlterTable) error {
	table := db.Table(stmt.Name)
	if table == nil {
		return fmt.Errorf("%w: table %s", ErrNotFound, stmt.Name)
	}
	switch stmt.Op {
	case sql.AlterKindAdd:
		return db.addColumn(table, stmt.Add)
	case sql.AlterKindDrop:
		return db.dropColumn(table, stmt.ColumnName)
	case sql.AlterKindType:
		col := table.Column(stmt.ColumnName)
		if col == nil {
			return fmt.Errorf("%w: column %s", ErrNotFound, stmt.ColumnName)
		}
		if !stmt.To.IsValid() {
			return fmt.Errorf("%w: column %s type", ErrInvalid, stmt.ColumnName)
		}
		col.Type = stmt.To
		return nil
	case sql.AlterKindRename:
		return db.renameTable(table, stmt.NewName)
	case sql.AlterKindRenameColumn:
		return db.renameColumn(table, stmt.ColumnName, stmt.NewName)
	default:
		return fmt.Errorf("%w: alter table", ErrInvalid)
	}
}

func (db *Database) addColumn(table *Table, col *sql.Column) error {
	if col == nil || col.Name == "" || !col.Type.IsValid() {
		return fmt.Errorf("%w: column", ErrInvalid)
	}
	if table.Column(col.Name) != nil {
		return fmt.Errorf("%w: column %s", ErrDuplicate, col.Name)
	}
	if col.Primary && len(table.Primary) > 0 {
		return fmt.Errorf("%w: primary key", ErrInvalid)
	}
	table.Columns = append(table.Columns, &Column{
		Name:     col.Name,
		Type:     col.Type,
		Primary:  col.Primary,
		Auto:     col.Auto,
		NotNull:  col.Required,
		IsUnique: col.IsUnique,
		Default:  cloneExpr(col.DefaultExpr),
	})
	if col.Primary {
		table.Primary = []string{col.Name}
	}
	if col.RefTable != "" {
		table.Foreign = append(table.Foreign, &ForeignKey{
			Columns:    []string{col.Name},
			Table:      col.RefTable,
			RefColumns: copyStrings(col.RefColumns),
			OnDelete:   col.DeleteAction,
			OnUpdate:   col.UpdateAction,
		})
	}
	return nil
}

func (db *Database) dropColumn(table *Table, name string) error {
	if table.Column(name) == nil {
		return fmt.Errorf("%w: column %s", ErrNotFound, name)
	}
	if db.columnReferenced(table.Name, name) {
		return fmt.Errorf("%w: column %s", ErrInvalid, name)
	}
	var cols []*Column
	for _, col := range table.Columns {
		if col.Name != name {
			cols = append(cols, col)
		}
	}
	table.Columns = cols
	table.Primary = removeString(table.Primary, name)
	for _, col := range table.Columns {
		col.Primary = contains(table.Primary, col.Name)
	}
	var uniques [][]string
	for _, set := range table.Uniques {
		if contains(set, name) {
			continue
		}
		uniques = append(uniques, set)
	}
	table.Uniques = uniques
	var fks []*ForeignKey
	for _, fk := range table.Foreign {
		if contains(fk.Columns, name) {
			continue
		}
		fks = append(fks, fk)
	}
	table.Foreign = fks
	var indexes []*Index
	for _, idx := range table.Indexes {
		if contains(idx.Columns, name) {
			delete(db.indexes, idx.Name)
			continue
		}
		indexes = append(indexes, idx)
	}
	table.Indexes = indexes
	return nil
}

func (db *Database) columnReferenced(table, column string) bool {
	for _, other := range db.Tables {
		for _, fk := range other.Foreign {
			if fk.Table == table && contains(fk.RefColumns, column) {
				return true
			}
		}
	}
	return false
}

func (db *Database) renameTable(table *Table, name string) error {
	if name == "" {
		return fmt.Errorf("%w: table name", ErrInvalid)
	}
	if name == table.Name {
		return nil
	}
	if _, ok := db.byName[name]; ok {
		return fmt.Errorf("%w: table %s", ErrDuplicate, name)
	}
	old := table.Name
	delete(db.byName, old)
	table.Name = name
	db.byName[name] = table
	for _, other := range db.Tables {
		for _, fk := range other.Foreign {
			if fk.Table == old {
				fk.Table = name
			}
		}
	}
	for index, owner := range db.indexes {
		if owner == old {
			db.indexes[index] = name
		}
	}
	return nil
}

func (db *Database) renameColumn(table *Table, from, to string) error {
	col := table.Column(from)
	if col == nil {
		return fmt.Errorf("%w: column %s", ErrNotFound, from)
	}
	if to == "" {
		return fmt.Errorf("%w: column name", ErrInvalid)
	}
	if to != from && table.Column(to) != nil {
		return fmt.Errorf("%w: column %s", ErrDuplicate, to)
	}
	col.Name = to
	table.Primary = replaceString(table.Primary, from, to)
	for i, set := range table.Uniques {
		table.Uniques[i] = replaceString(set, from, to)
	}
	for _, fk := range table.Foreign {
		fk.Columns = replaceString(fk.Columns, from, to)
	}
	for _, other := range db.Tables {
		for _, fk := range other.Foreign {
			if fk.Table == table.Name {
				fk.RefColumns = replaceString(fk.RefColumns, from, to)
			}
		}
	}
	for _, idx := range table.Indexes {
		idx.Columns = replaceString(idx.Columns, from, to)
	}
	return nil
}

func (db *Database) applyDrop(stmt *sql.DropTable) error {
	table := db.Table(stmt.Name)
	if table == nil {
		if stmt.MissingOK {
			return nil
		}
		return fmt.Errorf("%w: table %s", ErrNotFound, stmt.Name)
	}
	for _, other := range db.Tables {
		if other.Name == table.Name {
			continue
		}
		for _, fk := range other.Foreign {
			if fk.Table == table.Name {
				return fmt.Errorf("%w: %s references %s", ErrInvalid, other.Name, table.Name)
			}
		}
	}
	for _, idx := range table.Indexes {
		delete(db.indexes, idx.Name)
	}
	delete(db.byName, table.Name)
	var tables []*Table
	for _, item := range db.Tables {
		if item.Name != table.Name {
			tables = append(tables, item)
		}
	}
	db.Tables = tables
	return nil
}

func (db *Database) applyIndex(stmt *sql.CreateIndex) error {
	if stmt.Name == "" || stmt.Table == "" || len(stmt.Columns) == 0 {
		return fmt.Errorf("%w: create index", ErrInvalid)
	}
	table := db.Table(stmt.Table)
	if table == nil {
		return fmt.Errorf("%w: table %s", ErrNotFound, stmt.Table)
	}
	for _, name := range stmt.Columns {
		if table.Column(name) == nil {
			return fmt.Errorf("%w: index %s", ErrInvalid, name)
		}
	}
	if owner, ok := db.indexes[stmt.Name]; ok {
		if stmt.UnlessExists {
			return nil
		}
		return fmt.Errorf("%w: index %s", ErrDuplicate, owner+"."+stmt.Name)
	}
	idx := &Index{
		Name:     stmt.Name,
		IsUnique: stmt.IsUnique,
		Columns:  copyStrings(stmt.Columns),
		Where:    cloneExpr(stmt.Predicate),
	}
	table.Indexes = append(table.Indexes, idx)
	db.indexes[idx.Name] = table.Name
	return nil
}

// Make returns the CreateTable snapshot of the catalog, then its indexes.
func (db *Database) Make() (*sql.File, error) {
	if db == nil {
		return nil, fmt.Errorf("%w: catalog", ErrInvalid)
	}
	file := sql.NewFile()
	if db.Dialect == sql.Postgres {
		file.Postgres()
	}
	for _, table := range db.Tables {
		if err := writeTable(file, table); err != nil {
			return nil, err
		}
	}
	for _, table := range db.Tables {
		for _, idx := range table.Indexes {
			create := file.CreateIndex(idx.Name)
			if idx.IsUnique {
				create.Unique()
			}
			create.On(table.Name, idx.Columns...)
			if idx.Where != nil {
				create.Where(cloneExprValue(*idx.Where))
			}
		}
	}
	return file, nil
}

func writeTable(file *sql.File, table *Table) error {
	stmt := file.CreateTable(table.Name)
	inline := len(table.Primary) == 1
	used := map[string]bool{}
	for _, col := range table.Columns {
		out := stmt.Column(col.Name, col.Type)
		if inline && col.Name == table.Primary[0] {
			out.PrimaryKey()
		}
		if col.Auto {
			out.Autoincrement()
		}
		if col.NotNull {
			out.NotNull()
		}
		if col.IsUnique {
			out.Unique()
		}
		if col.Default != nil {
			out.Default(cloneExprValue(*col.Default))
		}
	}
	if !inline && len(table.Primary) > 0 {
		stmt.PrimaryKey(table.Primary...)
	}
	for _, cols := range table.Uniques {
		stmt.Unique(cols...)
	}
	for _, fk := range table.Foreign {
		if len(fk.Columns) == 1 && len(fk.RefColumns) == 1 && !used[fk.Columns[0]] {
			used[fk.Columns[0]] = true
			col := columnOf(stmt, fk.Columns[0])
			if col == nil {
				return fmt.Errorf("%w: foreign key %s", ErrInvalid, fk.Columns[0])
			}
			col.References(fk.Table, fk.RefColumns[0])
			if fk.OnDelete.IsValid() {
				col.OnDelete(fk.OnDelete)
			}
			if fk.OnUpdate.IsValid() {
				col.OnUpdate(fk.OnUpdate)
			}
			continue
		}
		key := stmt.ForeignKey(fk.Columns...)
		key.References(fk.Table, fk.RefColumns...)
		if fk.OnDelete.IsValid() {
			key.OnDelete(fk.OnDelete)
		}
		if fk.OnUpdate.IsValid() {
			key.OnUpdate(fk.OnUpdate)
		}
	}
	for _, check := range table.Checks {
		stmt.Check(cloneExprValue(check))
	}
	return nil
}

func columnOf(stmt *sql.CreateTable, name string) *sql.Column {
	for _, col := range stmt.Columns {
		if col.Name == name {
			return col
		}
	}
	return nil
}

func cloneExpr(expr *sql.Expr) *sql.Expr {
	if expr == nil {
		return nil
	}
	out := cloneExprValue(*expr)
	return &out
}

func cloneExprValue(expr sql.Expr) sql.Expr {
	if len(expr.Args) == 0 {
		return expr
	}
	args := make([]sql.Expr, len(expr.Args))
	for i := range expr.Args {
		args[i] = cloneExprValue(expr.Args[i])
	}
	expr.Args = args
	return expr
}

func copyStrings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return append([]string(nil), in...)
}

func removeString(in []string, name string) []string {
	var out []string
	for _, item := range in {
		if item != name {
			out = append(out, item)
		}
	}
	return copyStrings(out)
}

func replaceString(in []string, from, to string) []string {
	out := append([]string(nil), in...)
	for i, item := range out {
		if item == from {
			out[i] = to
		}
	}
	return copyStrings(out)
}

func contains(in []string, name string) bool {
	for _, item := range in {
		if item == name {
			return true
		}
	}
	return false
}
