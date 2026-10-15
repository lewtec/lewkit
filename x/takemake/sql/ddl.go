package sql

// CreateTable is a CREATE TABLE statement.
type CreateTable struct {
	note
	// Name is the table name.
	Name string
	// UnlessExists skips the statement when the table is already there.
	UnlessExists bool
	// Columns is the column list in order.
	Columns []*Column
	// Key is a table-level primary key. Column.Primary is the inline form.
	Key []string
	// Uniques is table-level UNIQUE constraints.
	Uniques [][]string
	// Foreign is table-level foreign keys.
	Foreign []*ForeignKey
	// Checks is table-level CHECK expressions.
	Checks []Expr
	file   *File
}

// Kind is CreateTable.
func (*CreateTable) Kind() Kind { return KindCreateTable }

func (*CreateTable) statement() {}

// CreateTable appends a CREATE TABLE statement.
func (f *File) CreateTable(name string) *CreateTable {
	stmt := &CreateTable{Name: name, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// IfNotExists marks the statement conditional.
func (t *CreateTable) IfNotExists() *CreateTable {
	if t != nil {
		t.UnlessExists = true
	}
	return t
}

// Column appends a column.
func (t *CreateTable) Column(name string, typ Type) *Column {
	col := &Column{Name: name, Type: typ, create: t}
	if t != nil {
		t.Columns = append(t.Columns, col)
	}
	return col
}

// PrimaryKey sets the table primary key.
// With no names it marks the last column as the primary key.
func (t *CreateTable) PrimaryKey(columns ...string) *CreateTable {
	if t == nil {
		return nil
	}
	if len(columns) == 0 {
		if n := len(t.Columns); n > 0 {
			t.Columns[n-1].Primary = true
		}
		return t
	}
	t.Key = append([]string(nil), columns...)
	return t
}

// Unique appends a table-level UNIQUE constraint.
func (t *CreateTable) Unique(columns ...string) *CreateTable {
	if t != nil && len(columns) > 0 {
		t.Uniques = append(t.Uniques, append([]string(nil), columns...))
	}
	return t
}

// ForeignKey appends a table-level foreign key on columns.
func (t *CreateTable) ForeignKey(columns ...string) *ForeignKey {
	fk := &ForeignKey{Columns: append([]string(nil), columns...), create: t}
	if t != nil {
		t.Foreign = append(t.Foreign, fk)
	}
	return fk
}

// Check appends a CHECK expression.
func (t *CreateTable) Check(expr Expr) *CreateTable {
	if t != nil {
		t.Checks = append(t.Checks, expr)
	}
	return t
}

// Column is one column in a CREATE TABLE or an ALTER TABLE ADD COLUMN.
type Column struct {
	// Name is the column name.
	Name string
	// Type is the column type.
	Type Type
	// Primary means the column is declared PRIMARY KEY.
	Primary bool
	// Auto means AUTOINCREMENT.
	Auto bool
	// Required means NOT NULL.
	Required bool
	// IsUnique means UNIQUE on this column.
	IsUnique bool
	// DefaultExpr is the DEFAULT expression.
	DefaultExpr *Expr
	// RefTable is the referenced table. Empty means no reference.
	RefTable string
	// RefColumns are the referenced columns.
	RefColumns []string
	// DeleteAction is the ON DELETE action. Zero means it was not set.
	DeleteAction Action
	// UpdateAction is the ON UPDATE action. Zero means it was not set.
	UpdateAction Action
	create       *CreateTable
	alter        *AlterTable
}

// Column starts the next column on the same table or alter.
func (c *Column) Column(name string, typ Type) *Column {
	if c == nil {
		return &Column{Name: name, Type: typ}
	}
	if c.create != nil {
		return c.create.Column(name, typ)
	}
	if c.alter != nil && c.alter.file != nil {
		return c.alter.file.AlterTable(c.alter.Name).AddColumn(name, typ)
	}
	return &Column{Name: name, Type: typ}
}

// PrimaryKey marks this column as the primary key when columns is empty.
// Names set the table-level primary key of the enclosing CREATE TABLE.
func (c *Column) PrimaryKey(columns ...string) *Column {
	if c == nil {
		return nil
	}
	if len(columns) == 0 {
		c.Primary = true
		return c
	}
	if c.create != nil {
		c.create.Key = append([]string(nil), columns...)
	}
	return c
}

// NotNull marks the column NOT NULL.
func (c *Column) NotNull() *Column {
	if c != nil {
		c.Required = true
	}
	return c
}

// Unique marks the column UNIQUE.
func (c *Column) Unique() *Column {
	if c != nil {
		c.IsUnique = true
	}
	return c
}

// Auto marks the column AUTOINCREMENT.
func (c *Column) Autoincrement() *Column {
	if c != nil {
		c.Auto = true
	}
	return c
}

// Default sets the DEFAULT expression.
func (c *Column) Default(expr Expr) *Column {
	if c != nil {
		c.DefaultExpr = &expr
	}
	return c
}

// References sets a single-column foreign key.
func (c *Column) References(table, column string) *Column {
	if c != nil {
		c.RefTable = table
		c.RefColumns = []string{column}
	}
	return c
}

// OnDelete sets the ON DELETE action.
func (c *Column) OnDelete(action Action) *Column {
	if c != nil {
		c.DeleteAction = action
	}
	return c
}

// OnUpdate sets the ON UPDATE action.
func (c *Column) OnUpdate(action Action) *Column {
	if c != nil {
		c.UpdateAction = action
	}
	return c
}

// ForeignKey appends a table-level foreign key on the enclosing CREATE TABLE.
func (c *Column) ForeignKey(columns ...string) *ForeignKey {
	if c == nil || c.create == nil {
		return &ForeignKey{Columns: append([]string(nil), columns...)}
	}
	return c.create.ForeignKey(columns...)
}

// ForeignKey is a table-level foreign key.
type ForeignKey struct {
	// Columns are the local columns.
	Columns []string
	// RefTable is the referenced table.
	RefTable string
	// RefColumns are the referenced columns.
	RefColumns []string
	// DeleteAction is the ON DELETE action.
	DeleteAction Action
	// UpdateAction is the ON UPDATE action.
	UpdateAction Action
	create       *CreateTable
}

// References sets the referenced table and columns.
func (fk *ForeignKey) References(table string, columns ...string) *ForeignKey {
	if fk != nil {
		fk.RefTable = table
		fk.RefColumns = append([]string(nil), columns...)
	}
	return fk
}

// OnDelete sets the ON DELETE action.
func (fk *ForeignKey) OnDelete(action Action) *ForeignKey {
	if fk != nil {
		fk.DeleteAction = action
	}
	return fk
}

// OnUpdate sets the ON UPDATE action.
func (fk *ForeignKey) OnUpdate(action Action) *ForeignKey {
	if fk != nil {
		fk.UpdateAction = action
	}
	return fk
}

// Column continues the CREATE TABLE with another column.
func (fk *ForeignKey) Column(name string, typ Type) *Column {
	if fk == nil || fk.create == nil {
		return &Column{Name: name, Type: typ}
	}
	return fk.create.Column(name, typ)
}

// AlterTable is one ALTER TABLE statement.
// A second change on the same value starts another statement.
type AlterTable struct {
	note
	// Name is the table being changed.
	Name string
	// Op is the change.
	Op AlterKind
	// Add is the column added when Op is add.
	Add *Column
	// ColumnName is the column dropped, retyped, or renamed.
	ColumnName string
	// To is the new type when Op retypes a column.
	To Type
	// NewName is the new table name or the new column name.
	NewName string
	file    *File
}

// Kind is AlterTable.
func (*AlterTable) Kind() Kind { return KindAlterTable }

func (*AlterTable) statement() {}

// AlterTable appends an ALTER TABLE statement.
func (f *File) AlterTable(name string) *AlterTable {
	stmt := &AlterTable{Name: name, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

func (a *AlterTable) next() *AlterTable {
	if a == nil {
		return &AlterTable{}
	}
	if a.file == nil {
		return &AlterTable{Name: a.Name}
	}
	return a.file.AlterTable(a.Name)
}

// AddColumn adds a column.
func (a *AlterTable) AddColumn(name string, typ Type) *Column {
	if a == nil {
		return &Column{Name: name, Type: typ}
	}
	if a.Op != 0 {
		return a.next().AddColumn(name, typ)
	}
	col := &Column{Name: name, Type: typ, alter: a}
	a.Op = AlterKindAdd
	a.Add = col
	return col
}

// DropColumn drops a column.
func (a *AlterTable) DropColumn(name string) *AlterTable {
	if a == nil {
		return nil
	}
	if a.Op != 0 {
		return a.next().DropColumn(name)
	}
	a.Op = AlterKindDrop
	a.ColumnName = name
	return a
}

// Type retypes a column.
func (a *AlterTable) Type(column string, typ Type) *AlterTable {
	if a == nil {
		return nil
	}
	if a.Op != 0 {
		return a.next().Type(column, typ)
	}
	a.Op = AlterKindType
	a.ColumnName = column
	a.To = typ
	return a
}

// Rename renames the table.
func (a *AlterTable) Rename(name string) *AlterTable {
	if a == nil {
		return nil
	}
	if a.Op != 0 {
		return a.next().Rename(name)
	}
	a.Op = AlterKindRename
	a.NewName = name
	return a
}

// RenameColumn renames a column.
func (a *AlterTable) RenameColumn(from, to string) *AlterTable {
	if a == nil {
		return nil
	}
	if a.Op != 0 {
		return a.next().RenameColumn(from, to)
	}
	a.Op = AlterKindRenameColumn
	a.ColumnName = from
	a.NewName = to
	return a
}

// DropTable is a DROP TABLE statement.
type DropTable struct {
	note
	// Name is the table.
	Name string
	// MissingOK skips a missing table.
	MissingOK bool
	// Cascading drops dependents.
	Cascading bool
	file      *File
}

// Kind is DropTable.
func (*DropTable) Kind() Kind { return KindDropTable }

func (*DropTable) statement() {}

// DropTable appends a DROP TABLE statement.
func (f *File) DropTable(name string) *DropTable {
	stmt := &DropTable{Name: name, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// IfExists marks the drop conditional.
func (d *DropTable) IfExists() *DropTable {
	if d != nil {
		d.MissingOK = true
	}
	return d
}

// Cascade marks the drop CASCADE.
func (d *DropTable) Cascade() *DropTable {
	if d != nil {
		d.Cascading = true
	}
	return d
}

// CreateIndex is a CREATE INDEX statement.
type CreateIndex struct {
	note
	// Name is the index name.
	Name string
	// IsUnique means CREATE UNIQUE INDEX.
	IsUnique bool
	// UnlessExists skips an index that already exists.
	UnlessExists bool
	// Table is the indexed table.
	Table string
	// Columns are the indexed columns in order.
	Columns []string
	// Predicate is a partial-index predicate.
	Predicate *Expr
	file      *File
}

// Kind is CreateIndex.
func (*CreateIndex) Kind() Kind { return KindCreateIndex }

func (*CreateIndex) statement() {}

// CreateIndex appends a CREATE INDEX statement.
func (f *File) CreateIndex(name string) *CreateIndex {
	stmt := &CreateIndex{Name: name, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// Unique marks the index unique.
func (idx *CreateIndex) Unique() *CreateIndex {
	if idx != nil {
		idx.IsUnique = true
	}
	return idx
}

// IfNotExists marks the statement conditional.
func (idx *CreateIndex) IfNotExists() *CreateIndex {
	if idx != nil {
		idx.UnlessExists = true
	}
	return idx
}

// On sets the table and columns.
func (idx *CreateIndex) On(table string, columns ...string) *CreateIndex {
	if idx != nil {
		idx.Table = table
		idx.Columns = append([]string(nil), columns...)
	}
	return idx
}

// Where sets a partial-index predicate.
func (idx *CreateIndex) Where(expr Expr) *CreateIndex {
	if idx != nil {
		idx.Predicate = &expr
	}
	return idx
}
