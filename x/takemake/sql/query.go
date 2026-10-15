package sql

// SelectItem is one item in a select list.
type SelectItem struct {
	// Expr is the value.
	Expr Expr
	// Alias is the AS name. Empty means no alias.
	Alias string
}

// OrderBy is one ORDER BY item.
type OrderBy struct {
	// Expr is the sort key.
	Expr Expr
	// Desc means DESC.
	Desc bool
}

// Select is a SELECT statement.
type Select struct {
	note
	// IsDistinct means SELECT DISTINCT.
	IsDistinct bool
	// Items is the select list.
	Items []SelectItem
	// Sources is the table list.
	Sources []string
	// Predicate is the predicate.
	Predicate *Expr
	// Groups is the GROUP BY list.
	Groups []Expr
	// Ordering is the ORDER BY list.
	Ordering []OrderBy
	// LimitExpr is the LIMIT expression.
	LimitExpr *Expr
	file      *File
}

// Kind is Select.
func (*Select) Kind() Kind { return KindSelect }

func (*Select) statement() {}

// Select appends a SELECT statement.
// Each expression is a select item. Item appends more.
func (f *File) Select(exprs ...Expr) *Select {
	stmt := &Select{file: f}
	for _, expr := range exprs {
		stmt.Items = append(stmt.Items, SelectItem{Expr: expr})
	}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// Distinct marks SELECT DISTINCT.
func (s *Select) Distinct() *Select {
	if s != nil {
		s.IsDistinct = true
	}
	return s
}

// Item appends a select item.
func (s *Select) Item(expr Expr) *Select {
	if s != nil {
		s.Items = append(s.Items, SelectItem{Expr: expr})
	}
	return s
}

// As sets the alias of the last select item.
func (s *Select) As(alias string) *Select {
	if s != nil && len(s.Items) > 0 {
		s.Items[len(s.Items)-1].Alias = alias
	}
	return s
}

// From appends tables.
func (s *Select) From(tables ...string) *Select {
	if s != nil {
		s.Sources = append(s.Sources, tables...)
	}
	return s
}

// Where sets the predicate.
func (s *Select) Where(expr Expr) *Select {
	if s != nil {
		s.Predicate = &expr
	}
	return s
}

// GroupBy appends GROUP BY expressions.
func (s *Select) GroupBy(exprs ...Expr) *Select {
	if s != nil {
		s.Groups = append(s.Groups, exprs...)
	}
	return s
}

// OrderBy appends sort keys.
func (s *Select) OrderBy(exprs ...Expr) *Select {
	if s != nil {
		for _, expr := range exprs {
			s.Ordering = append(s.Ordering, OrderBy{Expr: expr})
		}
	}
	return s
}

// Desc marks the last sort key DESC.
func (s *Select) Desc() *Select {
	if s != nil && len(s.Ordering) > 0 {
		s.Ordering[len(s.Ordering)-1].Desc = true
	}
	return s
}

// Limit sets the LIMIT expression.
func (s *Select) Limit(expr Expr) *Select {
	if s != nil {
		s.LimitExpr = &expr
	}
	return s
}

// Assignment is one SET target.
type Assignment struct {
	// Column is the column name.
	Column string
	// Value is the assigned expression.
	Value Expr
}

// Insert is an INSERT statement.
type Insert struct {
	note
	// Table is the target table.
	Table string
	// Targets is the column list. Empty means the default order.
	Targets []string
	// Rows is the VALUES list.
	Rows [][]Expr
	// Conflict means ON CONFLICT.
	Conflict bool
	// ConflictColumns are the conflict targets. Empty with Conflict means
	// the statement names no columns.
	ConflictColumns []string
	// Nothing means DO NOTHING.
	Nothing bool
	// Sets is DO UPDATE SET.
	Sets []Assignment
	file *File
}

// Kind is Insert.
func (*Insert) Kind() Kind { return KindInsert }

func (*Insert) statement() {}

// Insert appends an INSERT statement.
func (f *File) Insert(table string) *Insert {
	stmt := &Insert{Table: table, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// Columns sets the insert column list.
func (ins *Insert) Columns(names ...string) *Insert {
	if ins != nil {
		ins.Targets = append(ins.Targets, names...)
	}
	return ins
}

// Values appends one row.
func (ins *Insert) Values(exprs ...Expr) *Insert {
	if ins != nil {
		ins.Rows = append(ins.Rows, append([]Expr(nil), exprs...))
	}
	return ins
}

// OnConflict marks ON CONFLICT on columns.
func (ins *Insert) OnConflict(columns ...string) *Insert {
	if ins != nil {
		ins.Conflict = true
		ins.ConflictColumns = append([]string(nil), columns...)
	}
	return ins
}

// DoNothing marks DO NOTHING.
func (ins *Insert) DoNothing() *Insert {
	if ins != nil {
		ins.Conflict = true
		ins.Nothing = true
	}
	return ins
}

// Set appends one DO UPDATE assignment.
func (ins *Insert) Set(column string, value Expr) *Insert {
	if ins != nil {
		ins.Conflict = true
		ins.Sets = append(ins.Sets, Assignment{Column: column, Value: value})
	}
	return ins
}

// Update is an UPDATE statement.
type Update struct {
	note
	// Table is the target table.
	Table string
	// Sets are the assignments.
	Sets []Assignment
	// Predicate is the predicate.
	Predicate *Expr
	file      *File
}

// Kind is Update.
func (*Update) Kind() Kind { return KindUpdate }

func (*Update) statement() {}

// Update appends an UPDATE statement.
func (f *File) Update(table string) *Update {
	stmt := &Update{Table: table, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// Set appends an assignment.
func (u *Update) Set(column string, value Expr) *Update {
	if u != nil {
		u.Sets = append(u.Sets, Assignment{Column: column, Value: value})
	}
	return u
}

// Where sets the predicate.
func (u *Update) Where(expr Expr) *Update {
	if u != nil {
		u.Predicate = &expr
	}
	return u
}

// Delete is a DELETE statement.
type Delete struct {
	note
	// Table is the target table.
	Table string
	// Predicate is the predicate.
	Predicate *Expr
	file      *File
}

// Kind is Delete.
func (*Delete) Kind() Kind { return KindDelete }

func (*Delete) statement() {}

// Delete appends a DELETE statement.
func (f *File) Delete(table string) *Delete {
	stmt := &Delete{Table: table, file: f}
	if f != nil {
		f.Statements = append(f.Statements, stmt)
	}
	return stmt
}

// Where sets the predicate.
func (d *Delete) Where(expr Expr) *Delete {
	if d != nil {
		d.Predicate = &expr
	}
	return d
}
