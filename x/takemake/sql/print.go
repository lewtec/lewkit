package sql

import (
	"fmt"
	"strings"
	"unicode"
)

func formatFile(f *File) (string, error) {
	var b strings.Builder
	if f.Dialect.IsValid() {
		fmt.Fprintf(&b, "-- dialect: %s\n", f.Dialect.String())
	}
	for i, stmt := range f.Statements {
		if stmt == nil {
			return "", fmt.Errorf("%w: statement", ErrInvalid)
		}
		if i > 0 {
			b.WriteByte('\n')
		}
		if err := writeStmt(&b, stmt); err != nil {
			return "", err
		}
		b.WriteString(";\n")
	}
	return b.String(), nil
}

func writeStmt(b *strings.Builder, stmt Statement) error {
	switch stmt := stmt.(type) {
	case *CreateTable:
		writeComment(b, stmt.Comment)
		return writeCreateTable(b, stmt)
	case *AlterTable:
		writeComment(b, stmt.Comment)
		return writeAlter(b, stmt)
	case *DropTable:
		writeComment(b, stmt.Comment)
		return writeDrop(b, stmt)
	case *CreateIndex:
		writeComment(b, stmt.Comment)
		return writeIndex(b, stmt)
	case *Select:
		writeComment(b, stmt.Comment)
		return writeSelect(b, stmt)
	case *Insert:
		writeComment(b, stmt.Comment)
		return writeInsert(b, stmt)
	case *Update:
		writeComment(b, stmt.Comment)
		return writeUpdate(b, stmt)
	case *Delete:
		writeComment(b, stmt.Comment)
		return writeDelete(b, stmt)
	default:
		return fmt.Errorf("%w: statement", ErrInvalid)
	}
}

func writeComment(b *strings.Builder, comment string) {
	if comment == "" {
		return
	}
	for _, line := range strings.Split(comment, "\n") {
		b.WriteString("-- ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
}

func writeCreateTable(b *strings.Builder, stmt *CreateTable) error {
	if stmt.Name == "" {
		return fmt.Errorf("%w: table name", ErrInvalid)
	}
	b.WriteString("CREATE TABLE ")
	if stmt.UnlessExists {
		b.WriteString("IF NOT EXISTS ")
	}
	writeIdent(b, stmt.Name)
	b.WriteString(" (\n")
	var lines []string
	for _, col := range stmt.Columns {
		if col == nil {
			return fmt.Errorf("%w: column", ErrInvalid)
		}
		text, err := formatColumn(col)
		if err != nil {
			return err
		}
		lines = append(lines, text)
	}
	if len(stmt.Key) > 0 {
		lines = append(lines, "PRIMARY KEY ("+formatNames(stmt.Key)+")")
	}
	for _, cols := range stmt.Uniques {
		lines = append(lines, "UNIQUE ("+formatNames(cols)+")")
	}
	for _, fk := range stmt.Foreign {
		if fk == nil {
			return fmt.Errorf("%w: foreign key", ErrInvalid)
		}
		text, err := formatForeignKey(fk)
		if err != nil {
			return err
		}
		lines = append(lines, text)
	}
	for _, check := range stmt.Checks {
		var expr strings.Builder
		if err := writeExpr(&expr, check, 0); err != nil {
			return err
		}
		lines = append(lines, "CHECK ("+expr.String()+")")
	}
	for i, line := range lines {
		b.WriteString("    ")
		b.WriteString(line)
		if i+1 < len(lines) {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteByte(')')
	return nil
}

func formatColumn(col *Column) (string, error) {
	if col.Name == "" {
		return "", fmt.Errorf("%w: column name", ErrInvalid)
	}
	typeName := col.Type.SQL()
	if typeName == "" {
		return "", fmt.Errorf("%w: column %s type", ErrInvalid, col.Name)
	}
	var b strings.Builder
	writeIdent(&b, col.Name)
	b.WriteByte(' ')
	b.WriteString(typeName)
	if col.Primary {
		b.WriteString(" PRIMARY KEY")
		if col.Auto {
			b.WriteString(" AUTOINCREMENT")
		}
	} else if col.Auto {
		b.WriteString(" AUTOINCREMENT")
	}
	if col.Required {
		b.WriteString(" NOT NULL")
	}
	if col.IsUnique {
		b.WriteString(" UNIQUE")
	}
	if col.DefaultExpr != nil {
		b.WriteString(" DEFAULT ")
		if err := writeExpr(&b, *col.DefaultExpr, 0); err != nil {
			return "", err
		}
	}
	if col.RefTable != "" {
		b.WriteString(" REFERENCES ")
		writeIdent(&b, col.RefTable)
		b.WriteByte('(')
		b.WriteString(formatNames(col.RefColumns))
		b.WriteByte(')')
	}
	if err := writeActions(&b, col.DeleteAction, col.UpdateAction); err != nil {
		return "", err
	}
	return b.String(), nil
}

func formatForeignKey(fk *ForeignKey) (string, error) {
	if fk.RefTable == "" || len(fk.Columns) == 0 {
		return "", fmt.Errorf("%w: foreign key", ErrInvalid)
	}
	var b strings.Builder
	b.WriteString("FOREIGN KEY (")
	b.WriteString(formatNames(fk.Columns))
	b.WriteString(") REFERENCES ")
	writeIdent(&b, fk.RefTable)
	b.WriteByte('(')
	b.WriteString(formatNames(fk.RefColumns))
	b.WriteByte(')')
	if err := writeActions(&b, fk.DeleteAction, fk.UpdateAction); err != nil {
		return "", err
	}
	return b.String(), nil
}

func writeActions(b *strings.Builder, onDelete, onUpdate Action) error {
	if onDelete != 0 {
		text := onDelete.SQL()
		if text == "" {
			return fmt.Errorf("%w: on delete", ErrInvalid)
		}
		b.WriteString(" ON DELETE ")
		b.WriteString(text)
	}
	if onUpdate != 0 {
		text := onUpdate.SQL()
		if text == "" {
			return fmt.Errorf("%w: on update", ErrInvalid)
		}
		b.WriteString(" ON UPDATE ")
		b.WriteString(text)
	}
	return nil
}

func formatNames(names []string) string {
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteString(", ")
		}
		writeIdent(&b, name)
	}
	return b.String()
}

func writeAlter(b *strings.Builder, stmt *AlterTable) error {
	if stmt.Name == "" || !stmt.Op.IsValid() {
		return fmt.Errorf("%w: alter table", ErrInvalid)
	}
	b.WriteString("ALTER TABLE ")
	writeIdent(b, stmt.Name)
	b.WriteByte(' ')
	switch stmt.Op {
	case AlterKindAdd:
		if stmt.Add == nil {
			return fmt.Errorf("%w: add column", ErrInvalid)
		}
		b.WriteString("ADD COLUMN ")
		text, err := formatColumn(stmt.Add)
		if err != nil {
			return err
		}
		b.WriteString(text)
	case AlterKindDrop:
		if stmt.ColumnName == "" {
			return fmt.Errorf("%w: drop column", ErrInvalid)
		}
		b.WriteString("DROP COLUMN ")
		writeIdent(b, stmt.ColumnName)
	case AlterKindType:
		if stmt.ColumnName == "" || stmt.To.SQL() == "" {
			return fmt.Errorf("%w: column type", ErrInvalid)
		}
		b.WriteString("ALTER COLUMN ")
		writeIdent(b, stmt.ColumnName)
		b.WriteString(" TYPE ")
		b.WriteString(stmt.To.SQL())
	case AlterKindRename:
		if stmt.NewName == "" {
			return fmt.Errorf("%w: rename table", ErrInvalid)
		}
		b.WriteString("RENAME TO ")
		writeIdent(b, stmt.NewName)
	case AlterKindRenameColumn:
		if stmt.ColumnName == "" || stmt.NewName == "" {
			return fmt.Errorf("%w: rename column", ErrInvalid)
		}
		b.WriteString("RENAME COLUMN ")
		writeIdent(b, stmt.ColumnName)
		b.WriteString(" TO ")
		writeIdent(b, stmt.NewName)
	default:
		return fmt.Errorf("%w: alter table", ErrInvalid)
	}
	return nil
}

func writeDrop(b *strings.Builder, stmt *DropTable) error {
	if stmt.Name == "" {
		return fmt.Errorf("%w: drop table", ErrInvalid)
	}
	b.WriteString("DROP TABLE ")
	if stmt.MissingOK {
		b.WriteString("IF EXISTS ")
	}
	writeIdent(b, stmt.Name)
	if stmt.Cascading {
		b.WriteString(" CASCADE")
	}
	return nil
}

func writeIndex(b *strings.Builder, stmt *CreateIndex) error {
	if stmt.Table == "" || len(stmt.Columns) == 0 {
		return fmt.Errorf("%w: create index", ErrInvalid)
	}
	b.WriteString("CREATE ")
	if stmt.IsUnique {
		b.WriteString("UNIQUE ")
	}
	b.WriteString("INDEX ")
	if stmt.UnlessExists {
		b.WriteString("IF NOT EXISTS ")
	}
	if stmt.Name != "" {
		writeIdent(b, stmt.Name)
		b.WriteByte(' ')
	}
	b.WriteString("ON ")
	writeIdent(b, stmt.Table)
	b.WriteString(" (")
	b.WriteString(formatNames(stmt.Columns))
	b.WriteByte(')')
	if stmt.Predicate != nil {
		b.WriteString("\nWHERE ")
		if err := writeExpr(b, *stmt.Predicate, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeSelect(b *strings.Builder, stmt *Select) error {
	if len(stmt.Items) == 0 {
		return fmt.Errorf("%w: select list", ErrInvalid)
	}
	b.WriteString("SELECT ")
	if stmt.IsDistinct {
		b.WriteString("DISTINCT ")
	}
	for i, item := range stmt.Items {
		if i > 0 {
			b.WriteString(", ")
		}
		if err := writeExpr(b, item.Expr, 0); err != nil {
			return err
		}
		if item.Alias != "" {
			b.WriteString(" AS ")
			writeIdent(b, item.Alias)
		}
	}
	if len(stmt.Sources) > 0 {
		b.WriteString("\nFROM ")
		b.WriteString(formatNames(stmt.Sources))
	}
	if stmt.Predicate != nil {
		b.WriteString("\nWHERE ")
		if err := writeExpr(b, *stmt.Predicate, 0); err != nil {
			return err
		}
	}
	if len(stmt.Groups) > 0 {
		b.WriteString("\nGROUP BY ")
		if err := writeExprList(b, stmt.Groups); err != nil {
			return err
		}
	}
	if len(stmt.Ordering) > 0 {
		b.WriteString("\nORDER BY ")
		for i, item := range stmt.Ordering {
			if i > 0 {
				b.WriteString(", ")
			}
			if err := writeExpr(b, item.Expr, 0); err != nil {
				return err
			}
			if item.Desc {
				b.WriteString(" DESC")
			}
		}
	}
	if stmt.LimitExpr != nil {
		b.WriteString("\nLIMIT ")
		if err := writeExpr(b, *stmt.LimitExpr, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeInsert(b *strings.Builder, stmt *Insert) error {
	if stmt.Table == "" || len(stmt.Rows) == 0 {
		return fmt.Errorf("%w: insert", ErrInvalid)
	}
	if stmt.Nothing && len(stmt.Sets) > 0 {
		return fmt.Errorf("%w: on conflict", ErrInvalid)
	}
	b.WriteString("INSERT INTO ")
	writeIdent(b, stmt.Table)
	if len(stmt.Targets) > 0 {
		b.WriteString(" (")
		b.WriteString(formatNames(stmt.Targets))
		b.WriteByte(')')
	}
	b.WriteString("\nVALUES ")
	for i, row := range stmt.Rows {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteByte('(')
		if err := writeExprList(b, row); err != nil {
			return err
		}
		b.WriteByte(')')
	}
	if stmt.Conflict {
		b.WriteString("\nON CONFLICT")
		if len(stmt.ConflictColumns) > 0 {
			b.WriteString(" (")
			b.WriteString(formatNames(stmt.ConflictColumns))
			b.WriteByte(')')
		}
		b.WriteString(" DO ")
		if stmt.Nothing {
			b.WriteString("NOTHING")
			return nil
		}
		if len(stmt.Sets) == 0 {
			return fmt.Errorf("%w: on conflict", ErrInvalid)
		}
		b.WriteString("UPDATE SET ")
		if err := writeSets(b, stmt.Sets); err != nil {
			return err
		}
	}
	return nil
}

func writeUpdate(b *strings.Builder, stmt *Update) error {
	if stmt.Table == "" || len(stmt.Sets) == 0 {
		return fmt.Errorf("%w: update", ErrInvalid)
	}
	b.WriteString("UPDATE ")
	writeIdent(b, stmt.Table)
	b.WriteString("\nSET ")
	if err := writeSets(b, stmt.Sets); err != nil {
		return err
	}
	if stmt.Predicate != nil {
		b.WriteString("\nWHERE ")
		if err := writeExpr(b, *stmt.Predicate, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeDelete(b *strings.Builder, stmt *Delete) error {
	if stmt.Table == "" {
		return fmt.Errorf("%w: delete", ErrInvalid)
	}
	b.WriteString("DELETE FROM ")
	writeIdent(b, stmt.Table)
	if stmt.Predicate != nil {
		b.WriteString("\nWHERE ")
		if err := writeExpr(b, *stmt.Predicate, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeSets(b *strings.Builder, sets []Assignment) error {
	for i, set := range sets {
		if i > 0 {
			b.WriteString(", ")
		}
		if set.Column == "" {
			return fmt.Errorf("%w: assignment", ErrInvalid)
		}
		writeIdent(b, set.Column)
		b.WriteString(" = ")
		if err := writeExpr(b, set.Value, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeExprList(b *strings.Builder, exprs []Expr) error {
	for i, expr := range exprs {
		if i > 0 {
			b.WriteString(", ")
		}
		if err := writeExpr(b, expr, 0); err != nil {
			return err
		}
	}
	return nil
}

func writeExpr(b *strings.Builder, expr Expr, parent int) error {
	switch expr.Kind {
	case ExprKindIdent:
		if expr.Name == "" {
			return fmt.Errorf("%w: identifier", ErrInvalid)
		}
		writeName(b, expr.Name)
	case ExprKindString:
		writeString(b, expr.Text)
	case ExprKindNumber, ExprKindPlaceholder:
		if expr.Text == "" {
			return fmt.Errorf("%w: literal", ErrInvalid)
		}
		b.WriteString(expr.Text)
	case ExprKindStar:
		b.WriteByte('*')
	case ExprKindCall:
		if expr.Name == "" {
			return fmt.Errorf("%w: call", ErrInvalid)
		}
		writeName(b, expr.Name)
		b.WriteByte('(')
		if err := writeExprList(b, expr.Args); err != nil {
			return err
		}
		b.WriteByte(')')
	case ExprKindUnary:
		if len(expr.Args) != 1 {
			return fmt.Errorf("%w: operator", ErrInvalid)
		}
		rank := precedence(expr.Name)
		if expr.Name == "NOT" {
			b.WriteString("NOT ")
		} else {
			b.WriteString(expr.Name)
		}
		return writeOperand(b, expr.Args[0], rank, true)
	case ExprKindBinary:
		if len(expr.Args) != 2 || expr.Name == "" {
			return fmt.Errorf("%w: operator", ErrInvalid)
		}
		rank := precedence(expr.Name)
		if err := writeOperand(b, expr.Args[0], rank, false); err != nil {
			return err
		}
		b.WriteByte(' ')
		b.WriteString(expr.Name)
		b.WriteByte(' ')
		return writeOperand(b, expr.Args[1], rank, true)
	default:
		return fmt.Errorf("%w: expression", ErrInvalid)
	}
	if parent < 0 {
		return fmt.Errorf("%w: expression", ErrInvalid)
	}
	return nil
}

func writeOperand(b *strings.Builder, expr Expr, parent int, right bool) error {
	rank := exprPrecedence(expr)
	if rank > 0 && (rank < parent || (right && rank == parent)) {
		b.WriteByte('(')
		if err := writeExpr(b, expr, 0); err != nil {
			return err
		}
		b.WriteByte(')')
		return nil
	}
	return writeExpr(b, expr, parent)
}

func exprPrecedence(expr Expr) int {
	switch expr.Kind {
	case ExprKindUnary, ExprKindBinary:
		return precedence(expr.Name)
	default:
		return 100
	}
}

func precedence(op string) int {
	switch op {
	case "OR":
		return 1
	case "AND":
		return 2
	case "NOT":
		return 3
	case "=", "<>", "!=", "<", ">", "<=", ">=", "LIKE", "IS", "IS NOT":
		return 4
	case "+", "-", "||":
		return 5
	case "*", "/", "%":
		return 6
	default:
		return 7
	}
}

func writeString(b *strings.Builder, text string) {
	b.WriteByte('\'')
	b.WriteString(strings.ReplaceAll(text, "'", "''"))
	b.WriteByte('\'')
}

func writeName(b *strings.Builder, name string) {
	parts := strings.Split(name, ".")
	for i, part := range parts {
		if i > 0 {
			b.WriteByte('.')
		}
		writeIdent(b, part)
	}
}

func writeIdent(b *strings.Builder, name string) {
	if plainIdent(name) && !keyword(name) {
		b.WriteString(name)
		return
	}
	b.WriteByte('"')
	b.WriteString(strings.ReplaceAll(name, `"`, `""`))
	b.WriteByte('"')
}

func plainIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !unicode.IsLetter(r) && r != '_' {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

func keyword(name string) bool {
	_, ok := keywords[strings.ToLower(name)]
	return ok
}

var keywords = map[string]struct{}{
	"action": {}, "add": {}, "all": {}, "alter": {}, "and": {}, "as": {},
	"asc": {}, "autoincrement": {}, "begin": {}, "by": {}, "cascade": {},
	"case": {}, "check": {}, "collate": {}, "column": {}, "commit": {},
	"conflict": {}, "constraint": {}, "create": {}, "cross": {}, "data": {},
	"default": {}, "delete": {}, "desc": {}, "distinct": {}, "do": {},
	"drop": {}, "else": {}, "end": {}, "except": {}, "exists": {}, "foreign": {},
	"from": {}, "group": {}, "having": {}, "if": {}, "index": {}, "inner": {},
	"insert": {}, "intersect": {}, "into": {}, "is": {}, "join": {}, "key": {},
	"left": {}, "like": {}, "limit": {}, "natural": {}, "not": {}, "nothing": {},
	"on": {}, "or": {}, "order": {}, "outer": {}, "primary": {}, "references": {},
	"rename": {}, "restrict": {}, "right": {}, "select": {}, "set": {},
	"table": {}, "then": {}, "type": {}, "union": {}, "unique": {}, "update": {},
	"using": {}, "values": {}, "view": {}, "when": {}, "where": {}, "with": {},
}
