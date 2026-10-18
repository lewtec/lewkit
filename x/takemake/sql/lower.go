package sql

import (
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
)

func lowerFile(src string, root xparse.SyntaxNode[Term, Rule], notes commentMap) (*File, error) {
	file := NewFile()
	lead := notes.by[notes.first]
	rest, dialect, ok := splitDialect(lead)
	if ok {
		file.Dialect = dialect
	} else {
		rest = lead
	}
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		stmtNode := statementNode(child)
		if !stmtNode.Valid() {
			continue
		}
		stmt, err := lowerStmt(src, file, stmtNode)
		if err != nil {
			return nil, err
		}
		comment := notes.by[child.Start()]
		if child.Start() == notes.first {
			comment = rest
		}
		setComment(stmt, comment)
		file.Statements = append(file.Statements, stmt)
	}
	return file, nil
}

func statementNode(n xparse.SyntaxNode[Term, Rule]) xparse.SyntaxNode[Term, Rule] {
	rule, ok := n.Rule()
	if !ok || rule != RuleStatement {
		return xparse.SyntaxNode[Term, Rule]{}
	}
	if n.ChildCount() != 1 {
		return xparse.SyntaxNode[Term, Rule]{}
	}
	return n.Child(0)
}

func lowerStmt(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (Statement, error) {
	rule, ok := n.Rule()
	if !ok {
		return nil, syntaxAt(n, "expected statement")
	}
	switch rule {
	case RuleCreateTable:
		return lowerCreateTable(src, file, n)
	case RuleCreateIndex:
		return lowerCreateIndex(src, file, n)
	case RuleAlter:
		return lowerAlter(src, file, n)
	case RuleDrop:
		return lowerDrop(src, file, n)
	case RuleSelect:
		return lowerSelect(src, file, n)
	case RuleInsert:
		return lowerInsert(src, file, n)
	case RuleUpdate:
		return lowerUpdate(src, file, n)
	case RuleDelete:
		return lowerDelete(src, file, n)
	default:
		return nil, syntaxAt(n, "expected statement")
	}
}

func lowerCreateTable(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*CreateTable, error) {
	stmt := &CreateTable{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleIfNotExists:
			stmt.UnlessExists = true
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Name = name
		case RuleTableItem:
			if err := lowerTableItem(src, stmt, child); err != nil {
				return nil, err
			}
		}
	}
	if stmt.Name == "" {
		return nil, syntaxAt(n, "expected name")
	}
	return stmt, nil
}

func lowerTableItem(src string, stmt *CreateTable, n xparse.SyntaxNode[Term, Rule]) error {
	if n.ChildCount() != 1 {
		return syntaxAt(n, "expected column")
	}
	child := n.Child(0)
	rule, ok := child.Rule()
	if !ok {
		return syntaxAt(n, "expected column")
	}
	switch rule {
	case RuleColumn:
		col, err := lowerColumn(src, child)
		if err != nil {
			return err
		}
		col.create = stmt
		stmt.Columns = append(stmt.Columns, col)
		return nil
	case RuleTableConstraint:
		return lowerTableConstraint(src, stmt, child)
	default:
		return syntaxAt(n, "expected column")
	}
}

func lowerColumn(src string, n xparse.SyntaxNode[Term, Rule]) (*Column, error) {
	col := &Column{}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			col.Name = name
		case RuleTypeName:
			valueType, err := lowerType(src, child)
			if err != nil {
				return nil, err
			}
			col.Type = valueType
		case RuleColumnConstraint:
			if err := lowerColConstraint(src, col, child); err != nil {
				return nil, err
			}
		}
	}
	if col.Name == "" || !col.Type.IsValid() {
		return nil, syntaxAt(n, "expected column")
	}
	return col, nil
}

func lowerColConstraint(src string, col *Column, n xparse.SyntaxNode[Term, Rule]) error {
	if onNode := childRule(n, RuleOnAction); onNode.Valid() {
		return applyOnAction(src, onNode, func(action Action, which string) {
			if which == "update" {
				col.UpdateAction = action
			} else {
				col.DeleteAction = action
			}
		})
	}
	switch {
	case hasTerm(n, TermPrimary):
		col.Primary = true
		if hasTerm(n, TermAutoincrement) {
			col.Auto = true
		}
	case hasTerm(n, TermNot):
		col.Required = true
	case hasTerm(n, TermUnique):
		col.IsUnique = true
	case hasTerm(n, TermAutoincrement):
		col.Auto = true
	case hasTerm(n, TermDefault):
		expr, err := lowerExpr(src, childRule(n, RuleUnary))
		if err != nil {
			return err
		}
		col.DefaultExpr = &expr
	case hasTerm(n, TermReferences):
		name, err := lowerName(src, childRule(n, RuleName))
		if err != nil {
			return err
		}
		cols, err := lowerParenNames(src, childRule(n, RuleParenthesizedNames))
		if err != nil {
			return err
		}
		col.RefTable = name
		col.RefColumns = cols
	}
	return nil
}

func lowerTableConstraint(src string, stmt *CreateTable, n xparse.SyntaxNode[Term, Rule]) error {
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleTableKey:
			cols, err := lowerParenNames(src, childRule(child, RuleParenthesizedNames))
			if err != nil {
				return err
			}
			stmt.Key = cols
		case RuleTableUnique:
			cols, err := lowerParenNames(src, childRule(child, RuleParenthesizedNames))
			if err != nil {
				return err
			}
			stmt.Uniques = append(stmt.Uniques, cols)
		case RuleTableForeign:
			fk, err := lowerForeign(src, child)
			if err != nil {
				return err
			}
			fk.create = stmt
			stmt.Foreign = append(stmt.Foreign, fk)
		case RuleTableCheck:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return err
			}
			stmt.Checks = append(stmt.Checks, expr)
		}
	}
	return nil
}

func lowerForeign(src string, n xparse.SyntaxNode[Term, Rule]) (*ForeignKey, error) {
	fk := &ForeignKey{}
	var parens []xparse.SyntaxNode[Term, Rule]
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleParenthesizedNames:
			parens = append(parens, child)
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			fk.RefTable = name
		case RuleOnAction:
			if err := applyOnAction(src, child, func(action Action, which string) {
				if which == "update" {
					fk.UpdateAction = action
				} else {
					fk.DeleteAction = action
				}
			}); err != nil {
				return nil, err
			}
		}
	}
	if len(parens) != 2 || fk.RefTable == "" {
		return nil, syntaxAt(n, "expected foreign key")
	}
	var err error
	fk.Columns, err = lowerParenNames(src, parens[0])
	if err != nil {
		return nil, err
	}
	fk.RefColumns, err = lowerParenNames(src, parens[1])
	if err != nil {
		return nil, err
	}
	return fk, nil
}

func lowerCreateIndex(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*CreateIndex, error) {
	stmt := &CreateIndex{file: file}
	sawOn := false
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if termIs(child, TermUnique) {
			stmt.IsUnique = true
			continue
		}
		if termIs(child, TermOn) {
			sawOn = true
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleIfNotExists:
			stmt.UnlessExists = true
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			if !sawOn {
				stmt.Name = name
				continue
			}
			stmt.Table = name
		case RuleParenthesizedNames:
			cols, err := lowerParenNames(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Columns = cols
		case RuleWhere:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return nil, err
			}
			stmt.Predicate = &expr
		}
	}
	if stmt.Table == "" || len(stmt.Columns) == 0 {
		return nil, syntaxAt(n, "expected index")
	}
	return stmt, nil
}

func lowerAlter(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*AlterTable, error) {
	stmt := &AlterTable{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Name = name
		case RuleAlterAction:
			if err := lowerAlterAction(src, stmt, child); err != nil {
				return nil, err
			}
		}
	}
	if stmt.Name == "" || !stmt.Op.IsValid() {
		return nil, syntaxAt(n, "expected alter action")
	}
	return stmt, nil
}

func lowerAlterAction(src string, stmt *AlterTable, n xparse.SyntaxNode[Term, Rule]) error {
	if n.ChildCount() != 1 {
		return syntaxAt(n, "expected alter action")
	}
	child := n.Child(0)
	rule, ok := child.Rule()
	if !ok {
		return syntaxAt(n, "expected alter action")
	}
	switch rule {
	case RuleAlterRename:
		stmt.Op = AlterKindRename
		name, err := lowerName(src, childRule(child, RuleName))
		if err != nil {
			return err
		}
		stmt.NewName = name
	case RuleAlterRenameColumn:
		stmt.Op = AlterKindRenameColumn
		names := childRules(child, RuleName)
		if len(names) != 2 {
			return syntaxAt(child, "expected name")
		}
		var err error
		stmt.ColumnName, err = lowerName(src, names[0])
		if err != nil {
			return err
		}
		stmt.NewName, err = lowerName(src, names[1])
		if err != nil {
			return err
		}
	case RuleAlterAdd:
		stmt.Op = AlterKindAdd
		col, err := lowerColumn(src, childRule(child, RuleColumn))
		if err != nil {
			return err
		}
		col.alter = stmt
		stmt.Add = col
	case RuleAlterDrop:
		stmt.Op = AlterKindDrop
		name, err := lowerName(src, childRule(child, RuleName))
		if err != nil {
			return err
		}
		stmt.ColumnName = name
	case RuleAlterType:
		stmt.Op = AlterKindType
		name, err := lowerName(src, childRule(child, RuleName))
		if err != nil {
			return err
		}
		valueType, err := lowerType(src, childRule(child, RuleTypeName))
		if err != nil {
			return err
		}
		stmt.ColumnName = name
		stmt.To = valueType
	default:
		return syntaxAt(n, "expected alter action")
	}
	return nil
}

func lowerDrop(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*DropTable, error) {
	stmt := &DropTable{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if termIs(child, TermCascade) {
			stmt.Cascading = true
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleIfExists:
			stmt.MissingOK = true
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Name = name
		}
	}
	if stmt.Name == "" {
		return nil, syntaxAt(n, "expected name")
	}
	return stmt, nil
}

func lowerSelect(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*Select, error) {
	stmt := &Select{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if termIs(child, TermDistinct) {
			stmt.IsDistinct = true
			continue
		}
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleSelectItem:
			item, err := lowerSelectItem(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Items = append(stmt.Items, item)
		case RuleFrom:
			names, err := lowerNameList(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Sources = append(stmt.Sources, names...)
		case RuleWhere:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return nil, err
			}
			stmt.Predicate = &expr
		case RuleGroup:
			exprs, err := lowerExprList(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Groups = append(stmt.Groups, exprs...)
		case RuleOrder:
			for j := 0; j < child.ChildCount(); j++ {
				itemNode := child.Child(j)
				got, ok := itemNode.Rule()
				if !ok || got != RuleOrderItem {
					continue
				}
				item, err := lowerOrderItem(src, itemNode)
				if err != nil {
					return nil, err
				}
				stmt.Ordering = append(stmt.Ordering, item)
			}
		case RuleLimit:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return nil, err
			}
			stmt.LimitExpr = &expr
		}
	}
	if len(stmt.Items) == 0 {
		return nil, syntaxAt(n, "expected expression")
	}
	return stmt, nil
}

func lowerSelectItem(src string, n xparse.SyntaxNode[Term, Rule]) (SelectItem, error) {
	item := SelectItem{}
	expr, err := lowerExpr(src, childRule(n, RuleExpression))
	if err != nil {
		return SelectItem{}, err
	}
	item.Expr = expr
	if nameNode := childRule(n, RuleName); nameNode.Valid() {
		alias, err := lowerName(src, nameNode)
		if err != nil {
			return SelectItem{}, err
		}
		item.Alias = alias
		return item, nil
	}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if term, ok := child.Term(); ok && term == TermAlias {
			item.Alias = child.Text()
		}
	}
	return item, nil
}

func lowerOrderItem(src string, n xparse.SyntaxNode[Term, Rule]) (OrderBy, error) {
	expr, err := lowerExpr(src, childRule(n, RuleExpression))
	if err != nil {
		return OrderBy{}, err
	}
	return OrderBy{Expr: expr, Desc: hasTerm(n, TermDesc)}, nil
}

func lowerInsert(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*Insert, error) {
	stmt := &Insert{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Table = name
		case RuleParenthesizedNames:
			cols, err := lowerParenNames(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Targets = cols
		case RuleRow:
			row, err := lowerExprList(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Rows = append(stmt.Rows, row)
		case RuleConflict:
			if err := lowerConflict(src, stmt, child); err != nil {
				return nil, err
			}
		}
	}
	if stmt.Table == "" || len(stmt.Rows) == 0 {
		return nil, syntaxAt(n, "expected insert")
	}
	return stmt, nil
}

func lowerConflict(src string, stmt *Insert, n xparse.SyntaxNode[Term, Rule]) error {
	stmt.Conflict = true
	if paren := childRule(n, RuleParenthesizedNames); paren.Valid() {
		cols, err := lowerParenNames(src, paren)
		if err != nil {
			return err
		}
		stmt.ConflictColumns = cols
	}
	if hasTerm(n, TermNothing) {
		stmt.Nothing = true
		return nil
	}
	sets, err := lowerAssigns(src, n)
	if err != nil {
		return err
	}
	stmt.Sets = sets
	return nil
}

func lowerUpdate(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*Update, error) {
	stmt := &Update{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Table = name
		case RuleWhere:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return nil, err
			}
			stmt.Predicate = &expr
		}
	}
	sets, err := lowerAssigns(src, n)
	if err != nil {
		return nil, err
	}
	stmt.Sets = sets
	if stmt.Table == "" || len(stmt.Sets) == 0 {
		return nil, syntaxAt(n, "expected update")
	}
	return stmt, nil
}

func lowerDelete(src string, file *File, n xparse.SyntaxNode[Term, Rule]) (*Delete, error) {
	stmt := &Delete{file: file}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleName:
			name, err := lowerName(src, child)
			if err != nil {
				return nil, err
			}
			stmt.Table = name
		case RuleWhere:
			expr, err := lowerExpr(src, childRule(child, RuleExpression))
			if err != nil {
				return nil, err
			}
			stmt.Predicate = &expr
		}
	}
	if stmt.Table == "" {
		return nil, syntaxAt(n, "expected name")
	}
	return stmt, nil
}

func lowerAssigns(src string, n xparse.SyntaxNode[Term, Rule]) ([]Assignment, error) {
	var sets []Assignment
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok || rule != RuleAssign {
			continue
		}
		names := childRules(child, RuleName)
		exprs := childRules(child, RuleExpression)
		if len(names) != 1 || len(exprs) != 1 {
			return nil, syntaxAt(child, "expected assignment")
		}
		column, err := lowerName(src, names[0])
		if err != nil {
			return nil, err
		}
		value, err := lowerExpr(src, exprs[0])
		if err != nil {
			return nil, err
		}
		sets = append(sets, Assignment{Column: column, Value: value})
	}
	return sets, nil
}

func lowerExprList(src string, n xparse.SyntaxNode[Term, Rule]) ([]Expr, error) {
	var out []Expr
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case RuleExpression:
			expr, err := lowerExpr(src, child)
			if err != nil {
				return nil, err
			}
			out = append(out, expr)
		}
	}
	return out, nil
}

func lowerExpr(src string, n xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	if !n.Valid() {
		return Expr{}, syntaxAt(xparse.SyntaxNode[Term, Rule]{}, "expected expression")
	}
	if term, ok := n.Term(); ok {
		return lowerExprTerm(src, n, term)
	}
	if litIs(n, "*") {
		return Star(), nil
	}
	rule, ok := n.Rule()
	if !ok {
		if litIs(n, "(") && n.ChildCount() == 0 {
			return Expr{}, syntaxAt(n, "expected expression")
		}
		return Expr{}, syntaxAt(n, "expected expression")
	}
	switch rule {
	case RuleCall:
		return lowerCall(src, n)
	case RuleQualifiedName:
		name, err := lowerQual(src, n)
		if err != nil {
			return Expr{}, err
		}
		return Id(name), nil
	case RuleName:
		name, err := lowerName(src, n)
		if err != nil {
			return Expr{}, err
		}
		return Id(name), nil
	}
	if n.ChildCount() == 1 {
		return lowerExpr(src, n.Child(0))
	}
	if litIs(n.Child(0), "(") && n.ChildCount() == 3 {
		return lowerExpr(src, n.Child(1))
	}
	switch n.ChildCount() {
	case 2:
		arg, err := lowerExpr(src, n.Child(1))
		if err != nil {
			return Expr{}, err
		}
		return unary(opName(n.Child(0)), arg), nil
	case 3:
		left, err := lowerExpr(src, n.Child(0))
		if err != nil {
			return Expr{}, err
		}
		if termIs(n.Child(1), TermIs) {
			return binary("IS", left, Id("NULL")), nil
		}
		right, err := lowerExpr(src, n.Child(2))
		if err != nil {
			return Expr{}, err
		}
		return binary(opName(n.Child(1)), left, right), nil
	case 4:
		left, err := lowerExpr(src, n.Child(0))
		if err != nil {
			return Expr{}, err
		}
		return binary("IS NOT", left, Id("NULL")), nil
	default:
		return Expr{}, syntaxAt(n, "expected expression")
	}
}

func lowerExprTerm(src string, n xparse.SyntaxNode[Term, Rule], term Term) (Expr, error) {
	switch term {
	case TermString:
		text, err := decodeString(n.Text())
		if err != nil {
			return Expr{}, syntaxAt(n, "unclosed string")
		}
		return Lit(text), nil
	case TermNumber:
		return Num(n.Text()), nil
	case TermPlaceholder:
		return Expr{Kind: ExprKindPlaceholder, Text: n.Text()}, nil
	case TermIdentifier, TermAlias:
		return Id(n.Text()), nil
	case TermQuoted:
		text, err := decodeQuoted(n.Text())
		if err != nil {
			return Expr{}, syntaxAt(n, "unclosed identifier")
		}
		return Id(text), nil
	case TermNull:
		return Id("NULL"), nil
	default:
		return Expr{}, syntaxAt(n, "expected expression")
	}
}

func lowerCall(src string, n xparse.SyntaxNode[Term, Rule]) (Expr, error) {
	name, err := lowerQual(src, childRule(n, RuleQualifiedName))
	if err != nil {
		return Expr{}, err
	}
	args, err := lowerExprList(src, n)
	if err != nil {
		return Expr{}, err
	}
	return Call(name, args...), nil
}

func lowerQual(src string, n xparse.SyntaxNode[Term, Rule]) (string, error) {
	names, err := lowerNameList(src, n)
	if err != nil {
		return "", err
	}
	if len(names) == 0 {
		return "", syntaxAt(n, "expected name")
	}
	return strings.Join(names, "."), nil
}

func lowerNameList(src string, n xparse.SyntaxNode[Term, Rule]) ([]string, error) {
	var names []string
	for _, child := range childRules(n, RuleName) {
		name, err := lowerName(src, child)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func lowerParenNames(src string, n xparse.SyntaxNode[Term, Rule]) ([]string, error) {
	names, err := lowerNameList(src, n)
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, syntaxAt(n, "expected name")
	}
	return names, nil
}

func lowerName(src string, n xparse.SyntaxNode[Term, Rule]) (string, error) {
	if !n.Valid() {
		return "", syntaxAt(xparse.SyntaxNode[Term, Rule]{}, "expected name")
	}
	if n.ChildCount() == 0 {
		if term, ok := n.Term(); ok {
			expr, err := lowerExprTerm(src, n, term)
			if err != nil {
				return "", err
			}
			if expr.Kind == ExprKindIdent {
				return expr.Name, nil
			}
		}
		return "", syntaxAt(n, "expected name")
	}
	return lowerName(src, n.Child(0))
}

func lowerType(src string, n xparse.SyntaxNode[Term, Rule]) (Type, error) {
	word := ""
	if n.ChildCount() == 1 {
		word = n.Child(0).Text()
	} else {
		word = n.Text()
	}
	switch strings.ToUpper(word) {
	case "INTEGER", "INT":
		return TypeInteger, nil
	case "TEXT":
		return TypeText, nil
	case "BLOB":
		return TypeBlob, nil
	case "REAL":
		return TypeReal, nil
	case "NUMERIC":
		return TypeNumeric, nil
	case "BIGINT":
		return TypeBigInt, nil
	case "BOOLEAN", "BOOL":
		return TypeBoolean, nil
	case "TIMESTAMP":
		return TypeTimestamp, nil
	default:
		if word == "" {
			return 0, syntaxAt(n, "expected type")
		}
		return 0, syntaxAt(n, "unknown type "+word)
	}
}

func applyOnAction(src string, n xparse.SyntaxNode[Term, Rule], set func(Action, string)) error {
	which := "delete"
	if hasTerm(n, TermUpdate) {
		which = "update"
	}
	action, err := lowerAction(src, childRule(n, RuleAction))
	if err != nil {
		return err
	}
	set(action, which)
	return nil
}

func lowerAction(src string, n xparse.SyntaxNode[Term, Rule]) (Action, error) {
	switch {
	case hasTerm(n, TermCascade):
		return Cascade, nil
	case hasTerm(n, TermRestrict):
		return Restrict, nil
	case hasTerm(n, TermNull):
		return SetNull, nil
	case hasTerm(n, TermDefault):
		return SetDefault, nil
	case hasTerm(n, TermAction):
		return NoAction, nil
	default:
		return 0, syntaxAt(n, "expected referential action")
	}
}

func opName(n xparse.SyntaxNode[Term, Rule]) string {
	if term, ok := n.Term(); ok {
		return term.String()
	}
	return n.Text()
}

func childRule(n xparse.SyntaxNode[Term, Rule], want Rule) xparse.SyntaxNode[Term, Rule] {
	rules := childRules(n, want)
	if len(rules) == 0 {
		return xparse.SyntaxNode[Term, Rule]{}
	}
	return rules[0]
}

func childRules(n xparse.SyntaxNode[Term, Rule], want Rule) []xparse.SyntaxNode[Term, Rule] {
	var out []xparse.SyntaxNode[Term, Rule]
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		got, ok := child.Rule()
		if ok && got == want {
			out = append(out, child)
		}
	}
	return out
}

func hasTerm(n xparse.SyntaxNode[Term, Rule], want Term) bool {
	for i := 0; i < n.ChildCount(); i++ {
		if termIs(n.Child(i), want) {
			return true
		}
	}
	return false
}

func termIs(n xparse.SyntaxNode[Term, Rule], want Term) bool {
	got, ok := n.Term()
	return ok && got == want
}

func litIs(n xparse.SyntaxNode[Term, Rule], text string) bool {
	return n.Valid() && n.Name() == text
}

func splitDialect(comment string) (string, Dialect, bool) {
	if comment == "" {
		return "", 0, false
	}
	line, tail, _ := strings.Cut(comment, "\n")
	key, val, found := strings.Cut(strings.TrimSpace(line), ":")
	if !found || !strings.EqualFold(strings.TrimSpace(key), "dialect") {
		return comment, 0, false
	}
	dialect, err := ParseDialect(strings.TrimSpace(val))
	if err != nil {
		return comment, 0, false
	}
	return strings.TrimSpace(tail), dialect, true
}
