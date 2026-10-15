package sql

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	up := NewFile()
	up.CreateTable("artifact").
		Column("name", Text()).PrimaryKey().
		Column("format", Text()).NotNull().
		Column("entry", Integer()).NotNull()
	up.CreateTable("section").
		Column("artifact", Text()).NotNull().References("artifact", "name").OnDelete(Cascade).
		Column("position", Integer()).NotNull().
		Column("name", Text()).NotNull().
		PrimaryKey("artifact", "position")

	first := roundTrip(t, mustMake(t, up))
	require.Equal(t, SQLite, first.Dialect)
	require.Len(t, first.Statements, 2)

	artifact := first.Statements[0].(*CreateTable)
	require.Equal(t, "artifact", artifact.Name)
	require.Len(t, artifact.Columns, 3)
	require.True(t, artifact.Columns[0].Primary)
	require.Equal(t, TypeText, artifact.Columns[0].Type)
	require.True(t, artifact.Columns[1].Required)
	require.True(t, artifact.Columns[2].Required)
	require.Empty(t, artifact.Key)

	section := first.Statements[1].(*CreateTable)
	require.Equal(t, []string{"artifact", "position"}, section.Key)
	require.Equal(t, "artifact", section.Columns[0].RefTable)
	require.Equal(t, []string{"name"}, section.Columns[0].RefColumns)
	require.Equal(t, Cascade, section.Columns[0].DeleteAction)
	require.True(t, section.Columns[0].Required)
}

func TestStatementRoundTrip(t *testing.T) {
	t.Parallel()
	file := NewFile().Postgres()
	file.AlterTable("artifact").AddColumn("note", Text()).NotNull()
	file.AlterTable("artifact").Type("entry", BigInt())
	file.AlterTable("artifact").DropColumn("mode")
	file.AlterTable("artifact").RenameColumn("path", "file")
	file.AlterTable("artifact").Rename("item")
	file.DropTable("item").IfExists().Cascade()
	file.CreateIndex("item_file").Unique().IfNotExists().On("item", "file").Where(IsNotNull(Id("file")))
	file.Insert("item").Columns("file").Values(Lit("a")).OnConflict("file").Set("file", Lit("b"))
	file.Update("item").Set("file", Lit("c")).Where(Eq(Id("file"), Lit("b")))
	file.Delete("item").Where(Eq(Id("file"), Lit("c")))
	file.Select(Id("album")).
		Item(Call("MIN", Id("artist"))).As("artist").
		Item(Call("COUNT", Star())).As("n").
		Distinct().
		From("item").
		Where(Or(Eq(Arg(), Lit("")), Like(Id("file"), Arg()))).
		GroupBy(Id("album")).
		OrderBy(Id("album")).Desc().
		Limit(Num("10"))

	got := roundTrip(t, mustMake(t, file))
	require.Equal(t, Postgres, got.Dialect)
	require.Equal(t, describe(file), describe(got))
}

func TestCorpusRoundTrip(t *testing.T) {
	t.Parallel()
	project := roundTrip(t, read(t, "../../../x/disasm/decomp/schema/sqlite/migrations/000001_project.up.sql"))
	require.Len(t, project.Statements, 10)
	artifact := project.Statements[0].(*CreateTable)
	require.Equal(t, "artifact", artifact.Name)
	require.Len(t, artifact.Columns, 8)
	require.True(t, artifact.Columns[0].Primary)
	require.Equal(t, TypeBlob, artifact.Columns[5].Type)
	require.NotNil(t, artifact.Columns[7].DefaultExpr)
	require.Equal(t, "", artifact.Columns[7].DefaultExpr.Text)

	section := project.Statements[1].(*CreateTable)
	require.Equal(t, []string{"artifact", "position"}, section.Key)
	require.Equal(t, Cascade, section.Columns[0].DeleteAction)

	instruction := project.Statements[4].(*CreateTable)
	require.Equal(t, []string{"artifact", "function_address", "position"}, instruction.Key)
	require.Len(t, instruction.Foreign, 1)
	require.Equal(t, "function", instruction.Foreign[0].RefTable)
	require.Equal(t, []string{"artifact", "address"}, instruction.Foreign[0].RefColumns)
	require.Equal(t, Cascade, instruction.Foreign[0].DeleteAction)

	down := roundTrip(t, read(t, "../../../x/disasm/decomp/schema/sqlite/migrations/000001_project.down.sql"))
	require.Len(t, down.Statements, 10)
	require.Equal(t, "image_string", down.Statements[0].(*DropTable).Name)
	require.Equal(t, "artifact", down.Statements[9].(*DropTable).Name)

	queries := roundTrip(t, read(t, "../../../x/disasm/decomp/schema/sqlite/query.sql"))
	require.Len(t, queries.Statements, 23)
	require.Equal(t, KindSelect, queries.Statements[0].Kind())
	require.Contains(t, queries.Statements[0].(*Select).Comment, "ListArtifacts")
	upsert := queries.Statements[19].(*Insert)
	require.True(t, upsert.Conflict)
	require.Equal(t, []string{"artifact", "address"}, upsert.ConflictColumns)
	require.Equal(t, "text", upsert.Sets[0].Column)
	require.Equal(t, KindDelete, queries.Statements[20].Kind())

	tracks := roundTrip(t, read(t, "../../../examples/app/music/musicdb/sqlite/tracks.sql"))
	require.Len(t, tracks.Statements, 3)
	list := tracks.Statements[1].(*Select)
	require.Equal(t, "artist", list.Items[1].Alias)
	require.Equal(t, []string{"album"}, names(list.Groups))
	require.Len(t, list.Ordering, 1)

	items := roundTrip(t, read(t, "../../../x/db/sqlite/testdata/sqlite/migrations/000001_items.up.sql"))
	require.Len(t, items.Statements, 2)
	require.Equal(t, "alpha", items.Statements[1].(*Insert).Rows[0][0].Text)

	music := roundTrip(t, read(t, "../../../examples/app/music/musicdb/sqlite/migrations/000001_tracks.up.sql"))
	path := music.Statements[0].(*CreateTable).Columns[1]
	require.True(t, path.Required)
	require.True(t, path.IsUnique)

	pg := roundTrip(t, read(t, "../../../x/db/generate/testdata/ok/postgres/migrations/000001_items.up.sql"))
	require.Equal(t, TypeBigInt, pg.Statements[0].(*CreateTable).Columns[0].Type)
}

func TestTakeErrors(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrInvalid)
	require.ErrorIs(t, err, errs.ErrInvalid)

	_, err = Take(strings.NewReader("NOPE"))
	require.ErrorIs(t, err, ErrSyntax)
	require.ErrorIs(t, err, errs.ErrSyntax)

	file := NewFile()
	require.ErrorIs(t, file.Make(nil), ErrInvalid)

	empty, err := Take(strings.NewReader(""))
	require.NoError(t, err)
	require.Equal(t, SQLite, empty.Dialect)
	require.Empty(t, empty.Statements)

	postgres, err := Take(strings.NewReader("-- dialect: postgres\n"))
	require.NoError(t, err)
	require.Equal(t, Postgres, postgres.Dialect)
}

func roundTrip(t *testing.T, src string) *File {
	t.Helper()
	first, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	text := mustMake(t, first)
	second, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, describe(first), describe(second))
	require.Equal(t, text, mustMake(t, second))
	return first
}

func mustMake(t *testing.T, file *File) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return buf.String()
}

func read(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Clean(rel))
	require.NoError(t, err)
	return string(body)
}

func names(exprs []Expr) []string {
	out := make([]string, len(exprs))
	for i, expr := range exprs {
		out[i] = expr.Name
	}
	return out
}

func describe(file *File) string {
	var b strings.Builder
	b.WriteString(file.Dialect.String())
	b.WriteByte('\n')
	for _, stmt := range file.Statements {
		describeStmt(&b, stmt)
		b.WriteByte('\n')
	}
	return b.String()
}

func describeStmt(b *strings.Builder, stmt Statement) {
	switch stmt := stmt.(type) {
	case *CreateTable:
		b.WriteString("create ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Name)
		if stmt.UnlessExists {
			b.WriteString(" if")
		}
		for _, col := range stmt.Columns {
			b.WriteByte(' ')
			describeColumn(b, col)
		}
		b.WriteString(" key=")
		b.WriteString(strings.Join(stmt.Key, ","))
		for _, set := range stmt.Uniques {
			b.WriteString(" unique=")
			b.WriteString(strings.Join(set, ","))
		}
		for _, fk := range stmt.Foreign {
			b.WriteString(" fk=")
			describeKey(b, fk.Columns, fk.RefTable, fk.RefColumns, fk.DeleteAction, fk.UpdateAction)
		}
		for _, check := range stmt.Checks {
			b.WriteString(" check=")
			describeExpr(b, check)
		}
	case *AlterTable:
		b.WriteString("alter ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Name)
		b.WriteByte(' ')
		b.WriteString(stmt.Op.String())
		b.WriteByte(' ')
		b.WriteString(stmt.ColumnName)
		b.WriteByte(' ')
		b.WriteString(stmt.NewName)
		b.WriteByte(' ')
		b.WriteString(stmt.To.SQL())
		if stmt.Add != nil {
			b.WriteByte(' ')
			describeColumn(b, stmt.Add)
		}
	case *DropTable:
		b.WriteString("drop ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Name)
		if stmt.MissingOK {
			b.WriteString(" if")
		}
		if stmt.Cascading {
			b.WriteString(" cascade")
		}
	case *CreateIndex:
		b.WriteString("index ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Name)
		b.WriteByte(' ')
		b.WriteString(stmt.Table)
		b.WriteByte(' ')
		b.WriteString(strings.Join(stmt.Columns, ","))
		if stmt.IsUnique {
			b.WriteString(" unique")
		}
		if stmt.UnlessExists {
			b.WriteString(" if")
		}
		if stmt.Predicate != nil {
			b.WriteByte(' ')
			describeExpr(b, *stmt.Predicate)
		}
	case *Select:
		b.WriteString("select ")
		b.WriteString(stmt.Comment)
		if stmt.IsDistinct {
			b.WriteString(" distinct")
		}
		for _, item := range stmt.Items {
			b.WriteByte(' ')
			describeExpr(b, item.Expr)
			if item.Alias != "" {
				b.WriteString(" as ")
				b.WriteString(item.Alias)
			}
		}
		b.WriteString(" from=")
		b.WriteString(strings.Join(stmt.Sources, ","))
		if stmt.Predicate != nil {
			b.WriteString(" where=")
			describeExpr(b, *stmt.Predicate)
		}
		for _, expr := range stmt.Groups {
			b.WriteString(" group=")
			describeExpr(b, expr)
		}
		for _, item := range stmt.Ordering {
			b.WriteString(" order=")
			describeExpr(b, item.Expr)
			if item.Desc {
				b.WriteString(" desc")
			}
		}
		if stmt.LimitExpr != nil {
			b.WriteString(" limit=")
			describeExpr(b, *stmt.LimitExpr)
		}
	case *Insert:
		b.WriteString("insert ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Table)
		b.WriteByte(' ')
		b.WriteString(strings.Join(stmt.Targets, ","))
		for _, row := range stmt.Rows {
			b.WriteString(" row=")
			for i, expr := range row {
				if i > 0 {
					b.WriteByte(',')
				}
				describeExpr(b, expr)
			}
		}
		if stmt.Conflict {
			b.WriteString(" conflict=")
			b.WriteString(strings.Join(stmt.ConflictColumns, ","))
		}
		if stmt.Nothing {
			b.WriteString(" nothing")
		}
		for _, set := range stmt.Sets {
			b.WriteString(" set ")
			b.WriteString(set.Column)
			b.WriteByte('=')
			describeExpr(b, set.Value)
		}
	case *Update:
		b.WriteString("update ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Table)
		for _, set := range stmt.Sets {
			b.WriteString(" set ")
			b.WriteString(set.Column)
			b.WriteByte('=')
			describeExpr(b, set.Value)
		}
		if stmt.Predicate != nil {
			b.WriteString(" where=")
			describeExpr(b, *stmt.Predicate)
		}
	case *Delete:
		b.WriteString("delete ")
		b.WriteString(stmt.Comment)
		b.WriteByte(' ')
		b.WriteString(stmt.Table)
		if stmt.Predicate != nil {
			b.WriteString(" where=")
			describeExpr(b, *stmt.Predicate)
		}
	default:
		b.WriteString("unknown")
	}
}

func describeColumn(b *strings.Builder, col *Column) {
	b.WriteString(col.Name)
	b.WriteByte(':')
	b.WriteString(col.Type.SQL())
	if col.Primary {
		b.WriteString(":pk")
	}
	if col.Auto {
		b.WriteString(":auto")
	}
	if col.Required {
		b.WriteString(":nn")
	}
	if col.IsUnique {
		b.WriteString(":uq")
	}
	if col.DefaultExpr != nil {
		b.WriteString(":default=")
		describeExpr(b, *col.DefaultExpr)
	}
	if col.RefTable != "" {
		b.WriteString(":ref=")
		describeKey(b, []string{col.Name}, col.RefTable, col.RefColumns, col.DeleteAction, col.UpdateAction)
	}
}

func describeKey(b *strings.Builder, cols []string, table string, ref []string, onDelete, onUpdate Action) {
	b.WriteString(strings.Join(cols, ","))
	b.WriteByte('>')
	b.WriteString(table)
	b.WriteByte('(')
	b.WriteString(strings.Join(ref, ","))
	b.WriteByte(')')
	if onDelete.IsValid() {
		b.WriteByte(' ')
		b.WriteString(onDelete.SQL())
	}
	if onUpdate.IsValid() {
		b.WriteByte(' ')
		b.WriteString(onUpdate.SQL())
	}
}

func describeExpr(b *strings.Builder, expr Expr) {
	b.WriteString(expr.Kind.String())
	b.WriteByte(':')
	b.WriteString(expr.Name)
	b.WriteByte(':')
	b.WriteString(expr.Text)
	if len(expr.Args) == 0 {
		return
	}
	b.WriteByte('(')
	for i, arg := range expr.Args {
		if i > 0 {
			b.WriteByte(',')
		}
		describeExpr(b, arg)
	}
	b.WriteByte(')')
}
