package sql

func (p *parser) statement() (Statement, error) {
	switch {
	case p.eat("create"):
		return p.create()
	case p.eat("alter"):
		return p.alter()
	case p.eat("drop"):
		return p.drop()
	case p.eat("select"):
		return p.selectStmt()
	case p.eat("insert"):
		return p.insert()
	case p.eat("update"):
		return p.update()
	case p.eat("delete"):
		return p.deleteStmt()
	default:
		return nil, p.fail("expected statement")
	}
}

func (p *parser) create() (Statement, error) {
	unique := p.eat("unique")
	switch {
	case p.eat("table"):
		if unique {
			return nil, p.fail("expected INDEX")
		}
		return p.createTable()
	case p.eat("index"):
		return p.createIndex(unique)
	default:
		return nil, p.fail("expected TABLE or INDEX")
	}
}

func (p *parser) createTable() (*CreateTable, error) {
	stmt := &CreateTable{file: p.file}
	if p.eat("if") {
		if err := p.want("not"); err != nil {
			return nil, err
		}
		if err := p.want("exists"); err != nil {
			return nil, err
		}
		stmt.UnlessExists = true
	}
	name, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt.Name = name
	if err := p.tableBody(stmt); err != nil {
		return nil, err
	}
	return stmt, nil
}

func (p *parser) tableBody(stmt *CreateTable) error {
	if err := p.wantOp("("); err != nil {
		return err
	}
	if p.eatOp(")") {
		return nil
	}
	for {
		if p.tableConstraintStart() {
			if err := p.tableConstraint(stmt); err != nil {
				return err
			}
		} else if err := p.columnDef(stmt); err != nil {
			return err
		}
		if p.eatOp(",") {
			if p.atOp(")") {
				return p.fail("expected column")
			}
			continue
		}
		break
	}
	return p.wantOp(")")
}

func (p *parser) tableConstraintStart() bool {
	if p.tok.quoted {
		return false
	}
	return p.keyword("constraint") || p.keyword("primary") || p.keyword("foreign") || p.keyword("unique") || p.keyword("check")
}

func (p *parser) columnDef(stmt *CreateTable) error {
	name, err := p.simpleName()
	if err != nil {
		return err
	}
	typ, err := p.sqlType()
	if err != nil {
		return err
	}
	col := &Column{Name: name, Type: typ, create: stmt}
	if err := p.columnConstraints(col); err != nil {
		return err
	}
	stmt.Columns = append(stmt.Columns, col)
	return nil
}

func (p *parser) columnConstraints(col *Column) error {
	for {
		switch {
		case p.eat("constraint"):
			// Names are not part of the model.
			if _, err := p.simpleName(); err != nil {
				return err
			}
		case p.eat("primary"):
			if err := p.want("key"); err != nil {
				return err
			}
			col.Primary = true
			if p.eat("autoincrement") {
				col.Auto = true
			}
			p.eat("asc")
			p.eat("desc")
		case p.eat("not"):
			if err := p.want("null"); err != nil {
				return err
			}
			col.Required = true
		case p.eat("null"):
		case p.eat("unique"):
			col.IsUnique = true
		case p.eat("autoincrement"):
			col.Auto = true
		case p.eat("default"):
			expr, err := p.unary()
			if err != nil {
				return err
			}
			col.DefaultExpr = &expr
		case p.eat("references"):
			table, cols, err := p.reference()
			if err != nil {
				return err
			}
			col.RefTable = table
			col.RefColumns = cols
		case p.keyword("on"):
			action, which, err := p.onAction()
			if err != nil {
				return err
			}
			if which == "update" {
				col.UpdateAction = action
			} else {
				col.DeleteAction = action
			}
		default:
			return nil
		}
	}
}

func (p *parser) reference() (string, []string, error) {
	table, err := p.simpleName()
	if err != nil {
		return "", nil, err
	}
	cols, err := p.parenNames()
	if err != nil {
		return "", nil, err
	}
	return table, cols, nil
}

func (p *parser) onAction() (Action, string, error) {
	if err := p.want("on"); err != nil {
		return 0, "", err
	}
	var which string
	switch {
	case p.eat("delete"):
		which = "delete"
	case p.eat("update"):
		which = "update"
	default:
		return 0, "", p.fail("expected DELETE or UPDATE")
	}
	action, err := p.action()
	if err != nil {
		return 0, "", err
	}
	return action, which, nil
}

func (p *parser) action() (Action, error) {
	switch {
	case p.eat("cascade"):
		return Cascade, nil
	case p.eat("restrict"):
		return Restrict, nil
	case p.eat("set"):
		switch {
		case p.eat("null"):
			return SetNull, nil
		case p.eat("default"):
			return SetDefault, nil
		default:
			return 0, p.fail("expected NULL or DEFAULT")
		}
	case p.eat("no"):
		if err := p.want("action"); err != nil {
			return 0, err
		}
		return NoAction, nil
	default:
		return 0, p.fail("expected referential action")
	}
}

func (p *parser) tableConstraint(stmt *CreateTable) error {
	if p.eat("constraint") {
		if _, err := p.simpleName(); err != nil {
			return err
		}
	}
	switch {
	case p.eat("primary"):
		if err := p.want("key"); err != nil {
			return err
		}
		cols, err := p.parenNames()
		if err != nil {
			return err
		}
		stmt.Key = cols
		return nil
	case p.eat("unique"):
		cols, err := p.parenNames()
		if err != nil {
			return err
		}
		stmt.Uniques = append(stmt.Uniques, cols)
		return nil
	case p.eat("foreign"):
		if err := p.want("key"); err != nil {
			return err
		}
		cols, err := p.parenNames()
		if err != nil {
			return err
		}
		if err := p.want("references"); err != nil {
			return err
		}
		table, refCols, err := p.reference()
		if err != nil {
			return err
		}
		fk := &ForeignKey{Columns: cols, RefTable: table, RefColumns: refCols, create: stmt}
		for p.keyword("on") {
			action, which, err := p.onAction()
			if err != nil {
				return err
			}
			if which == "update" {
				fk.UpdateAction = action
			} else {
				fk.DeleteAction = action
			}
		}
		stmt.Foreign = append(stmt.Foreign, fk)
		return nil
	case p.eat("check"):
		if err := p.wantOp("("); err != nil {
			return err
		}
		expr, err := p.expr()
		if err != nil {
			return err
		}
		if err := p.wantOp(")"); err != nil {
			return err
		}
		stmt.Checks = append(stmt.Checks, expr)
		return nil
	default:
		return p.fail("expected constraint")
	}
}

func (p *parser) createIndex(unique bool) (*CreateIndex, error) {
	stmt := &CreateIndex{IsUnique: unique, file: p.file}
	if p.eat("if") {
		if err := p.want("not"); err != nil {
			return nil, err
		}
		if err := p.want("exists"); err != nil {
			return nil, err
		}
		stmt.UnlessExists = true
	}
	if p.tok.kind == tokIdent && !p.keyword("on") {
		name, err := p.simpleName()
		if err != nil {
			return nil, err
		}
		stmt.Name = name
	}
	if err := p.want("on"); err != nil {
		return nil, err
	}
	table, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	cols, err := p.parenNames()
	if err != nil {
		return nil, err
	}
	stmt.Table = table
	stmt.Columns = cols
	if p.eat("where") {
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		stmt.Predicate = &expr
	}
	return stmt, nil
}

func (p *parser) alter() (*AlterTable, error) {
	if err := p.want("table"); err != nil {
		return nil, err
	}
	name, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt := &AlterTable{Name: name, file: p.file}
	switch {
	case p.eat("rename"):
		if p.eat("to") {
			stmt.Op = AlterKindRename
			stmt.NewName, err = p.simpleName()
			return stmt, err
		}
		p.eat("column")
		stmt.Op = AlterKindRenameColumn
		stmt.ColumnName, err = p.simpleName()
		if err != nil {
			return nil, err
		}
		if err := p.want("to"); err != nil {
			return nil, err
		}
		stmt.NewName, err = p.simpleName()
		return stmt, err
	case p.eat("add"):
		p.eat("column")
		col, err := p.looseColumn()
		if err != nil {
			return nil, err
		}
		col.alter = stmt
		stmt.Op = AlterKindAdd
		stmt.Add = col
		return stmt, nil
	case p.eat("drop"):
		p.eat("column")
		stmt.Op = AlterKindDrop
		stmt.ColumnName, err = p.simpleName()
		return stmt, err
	case p.eat("alter"):
		p.eat("column")
		stmt.Op = AlterKindType
		stmt.ColumnName, err = p.simpleName()
		if err != nil {
			return nil, err
		}
		if p.eat("set") {
			if err := p.want("data"); err != nil {
				return nil, err
			}
		}
		if err := p.want("type"); err != nil {
			return nil, err
		}
		stmt.To, err = p.sqlType()
		return stmt, err
	default:
		return nil, p.fail("expected alter action")
	}
}

func (p *parser) looseColumn() (*Column, error) {
	name, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	typ, err := p.sqlType()
	if err != nil {
		return nil, err
	}
	col := &Column{Name: name, Type: typ}
	if err := p.columnConstraints(col); err != nil {
		return nil, err
	}
	return col, nil
}

func (p *parser) drop() (*DropTable, error) {
	if err := p.want("table"); err != nil {
		return nil, err
	}
	stmt := &DropTable{file: p.file}
	if p.eat("if") {
		if err := p.want("exists"); err != nil {
			return nil, err
		}
		stmt.MissingOK = true
	}
	name, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt.Name = name
	if p.eat("cascade") {
		stmt.Cascading = true
	} else {
		p.eat("restrict")
	}
	return stmt, nil
}

func (p *parser) selectStmt() (*Select, error) {
	stmt := &Select{file: p.file}
	if p.eat("distinct") {
		stmt.IsDistinct = true
	}
	for {
		item, err := p.selectItem()
		if err != nil {
			return nil, err
		}
		stmt.Items = append(stmt.Items, item)
		if !p.eatOp(",") {
			break
		}
	}
	if p.eat("from") {
		for {
			name, err := p.simpleName()
			if err != nil {
				return nil, err
			}
			stmt.Sources = append(stmt.Sources, name)
			if !p.eatOp(",") {
				break
			}
		}
	}
	if p.eat("where") {
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		stmt.Predicate = &expr
	}
	if p.eat("group") {
		if err := p.want("by"); err != nil {
			return nil, err
		}
		for {
			expr, err := p.expr()
			if err != nil {
				return nil, err
			}
			stmt.Groups = append(stmt.Groups, expr)
			if !p.eatOp(",") {
				break
			}
		}
	}
	if p.eat("order") {
		if err := p.want("by"); err != nil {
			return nil, err
		}
		for {
			expr, err := p.expr()
			if err != nil {
				return nil, err
			}
			item := OrderBy{Expr: expr}
			if p.eat("desc") {
				item.Desc = true
			} else {
				p.eat("asc")
			}
			stmt.Ordering = append(stmt.Ordering, item)
			if !p.eatOp(",") {
				break
			}
		}
	}
	if p.eat("limit") {
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		stmt.LimitExpr = &expr
	}
	return stmt, nil
}

func (p *parser) selectItem() (SelectItem, error) {
	expr, err := p.expr()
	if err != nil {
		return SelectItem{}, err
	}
	item := SelectItem{Expr: expr}
	if p.eat("as") {
		alias, err := p.simpleName()
		if err != nil {
			return SelectItem{}, err
		}
		item.Alias = alias
		return item, nil
	}
	if p.tok.kind == tokIdent && !p.tok.quoted && !p.clauseKeyword() {
		item.Alias = p.tok.text
		p.next()
	}
	return item, nil
}

func (p *parser) clauseKeyword() bool {
	switch stringsLower(p.tok.text) {
	case "from", "where", "group", "order", "limit", "having", "union", "except", "intersect", "window":
		return true
	default:
		return false
	}
}

func stringsLower(text string) string {
	return asciiLower(text)
}

func asciiLower(text string) string {
	b := []byte(text)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}

func (p *parser) insert() (*Insert, error) {
	if err := p.want("into"); err != nil {
		return nil, err
	}
	table, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt := &Insert{Table: table, file: p.file}
	if p.eatOp("(") {
		cols, err := p.nameList()
		if err != nil {
			return nil, err
		}
		if err := p.wantOp(")"); err != nil {
			return nil, err
		}
		stmt.Targets = cols
	}
	if err := p.want("values"); err != nil {
		return nil, err
	}
	for {
		if err := p.wantOp("("); err != nil {
			return nil, err
		}
		var row []Expr
		for {
			expr, err := p.expr()
			if err != nil {
				return nil, err
			}
			row = append(row, expr)
			if !p.eatOp(",") {
				break
			}
		}
		if err := p.wantOp(")"); err != nil {
			return nil, err
		}
		stmt.Rows = append(stmt.Rows, row)
		if !p.eatOp(",") {
			break
		}
	}
	if p.eat("on") {
		if err := p.want("conflict"); err != nil {
			return nil, err
		}
		stmt.Conflict = true
		if p.eatOp("(") {
			cols, err := p.nameList()
			if err != nil {
				return nil, err
			}
			if err := p.wantOp(")"); err != nil {
				return nil, err
			}
			stmt.ConflictColumns = cols
		}
		if err := p.want("do"); err != nil {
			return nil, err
		}
		switch {
		case p.eat("nothing"):
			stmt.Nothing = true
		case p.eat("update"):
			if err := p.want("set"); err != nil {
				return nil, err
			}
			stmt.Sets, err = p.assignments()
			if err != nil {
				return nil, err
			}
		default:
			return nil, p.fail("expected NOTHING or UPDATE")
		}
	}
	return stmt, nil
}

func (p *parser) assignments() ([]Assignment, error) {
	var sets []Assignment
	for {
		column, err := p.simpleName()
		if err != nil {
			return nil, err
		}
		if err := p.wantOp("="); err != nil {
			return nil, err
		}
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		sets = append(sets, Assignment{Column: column, Value: expr})
		if !p.eatOp(",") {
			return sets, nil
		}
	}
}

func (p *parser) update() (*Update, error) {
	table, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt := &Update{Table: table, file: p.file}
	if err := p.want("set"); err != nil {
		return nil, err
	}
	stmt.Sets, err = p.assignments()
	if err != nil {
		return nil, err
	}
	if p.eat("where") {
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		stmt.Predicate = &expr
	}
	return stmt, nil
}

func (p *parser) deleteStmt() (*Delete, error) {
	if err := p.want("from"); err != nil {
		return nil, err
	}
	table, err := p.simpleName()
	if err != nil {
		return nil, err
	}
	stmt := &Delete{Table: table, file: p.file}
	if p.eat("where") {
		expr, err := p.expr()
		if err != nil {
			return nil, err
		}
		stmt.Predicate = &expr
	}
	return stmt, nil
}
