package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/lewtec/lewkit/x/takemake/sql"
	"github.com/stretchr/testify/require"
)

func TestApplyCompose(t *testing.T) {
	t.Parallel()
	up := sql.NewFile()
	up.CreateTable("artifact").
		Column("name", sql.Text()).PrimaryKey().
		Column("format", sql.Text()).NotNull().
		Column("entry", sql.Integer()).NotNull()
	up.CreateTable("section").
		Column("artifact", sql.Text()).NotNull().References("artifact", "name").OnDelete(sql.Cascade).
		Column("position", sql.Integer()).NotNull().
		Column("name", sql.Text()).NotNull().
		PrimaryKey("artifact", "position")

	db := New()
	require.NoError(t, db.Apply(up))
	require.Equal(t, sql.SQLite, db.Dialect)

	name := db.Table("artifact").Column("name")
	require.Equal(t, sql.TypeText, name.Type)
	require.True(t, name.Primary)

	section := db.Table("section")
	fk := section.ForeignKey("artifact")
	require.Equal(t, []string{"artifact"}, fk.Columns)
	require.Equal(t, []string{"name"}, fk.RefColumns)
	require.Equal(t, sql.Cascade, fk.OnDelete)
	require.Equal(t, []string{"artifact", "position"}, section.Primary)

	fresh := New()
	snapshot, err := db.Make()
	require.NoError(t, err)
	require.NoError(t, fresh.Apply(snapshot))
	require.Equal(t, describeDB(db), describeDB(fresh))
}

func TestMigrationSnapshot(t *testing.T) {
	t.Parallel()
	file, err := sql.Take(strings.NewReader(read(t, "../../../../x/disasm/decomp/schema/sqlite/migrations/000001_project.up.sql")))
	require.NoError(t, err)
	db := New()
	require.NoError(t, db.Apply(file))
	require.Len(t, db.Tables, 10)

	artifact := db.Table("artifact")
	require.True(t, artifact.Column("name").Primary)
	require.Equal(t, sql.TypeBlob, artifact.Column("image").Type)
	require.Equal(t, "", artifact.Column("digest").Default.Text)
	require.Equal(t, sql.Cascade, db.Table("section").ForeignKey("artifact").OnDelete)
	instruction := db.Table("instruction").ForeignKey("function")
	require.Equal(t, []string{"artifact", "function_address"}, instruction.Columns)
	require.Equal(t, []string{"artifact", "address"}, instruction.RefColumns)

	queries, err := sql.Take(strings.NewReader(read(t, "../../../../x/disasm/decomp/schema/sqlite/query.sql")))
	require.NoError(t, err)
	before := describeDB(db)
	require.NoError(t, db.Apply(queries))
	require.Equal(t, before, describeDB(db))

	snapshot, err := db.Make()
	require.NoError(t, err)
	fresh := New()
	require.NoError(t, fresh.Apply(snapshot))
	require.Equal(t, describeDB(db), describeDB(fresh))

	down, err := sql.Take(strings.NewReader(read(t, "../../../../x/disasm/decomp/schema/sqlite/migrations/000001_project.down.sql")))
	require.NoError(t, err)
	require.NoError(t, db.Apply(down))
	require.Empty(t, db.Tables)
}

func TestAlterDropAndIndex(t *testing.T) {
	t.Parallel()
	up := sql.NewFile()
	up.CreateTable("artifact").
		Column("name", sql.Text()).PrimaryKey().
		Column("entry", sql.Integer()).NotNull()
	up.CreateIndex("artifact_entry").On("artifact", "entry")
	up.AlterTable("artifact").AddColumn("format", sql.Text()).NotNull().Default(sql.Lit(""))
	up.AlterTable("artifact").Type("format", sql.Integer())
	up.AlterTable("artifact").RenameColumn("format", "kind")
	up.AlterTable("artifact").Rename("item")

	db := New()
	require.NoError(t, db.Apply(up))
	item := db.Table("item")
	require.NotNil(t, item)
	require.Nil(t, db.Table("artifact"))
	require.Equal(t, sql.TypeInteger, item.Column("kind").Type)
	require.True(t, item.Column("kind").NotNull)
	require.Equal(t, "artifact_entry", item.Index("artifact_entry").Name)

	drop := sql.NewFile()
	drop.AlterTable("item").DropColumn("kind")
	drop.DropTable("item")
	require.NoError(t, db.Apply(drop))
	require.Empty(t, db.Tables)

	fresh := New()
	snapshot, err := db.Make()
	require.NoError(t, err)
	require.NoError(t, fresh.Apply(snapshot))
	require.Empty(t, fresh.Tables)
}

func TestCatalogErrors(t *testing.T) {
	t.Parallel()
	db := New()
	require.ErrorIs(t, db.Apply(nil), ErrInvalid)
	require.ErrorIs(t, db.Apply(nil), errs.ErrInvalid)
	require.Nil(t, db.Table("missing"))

	up := sql.NewFile()
	up.CreateTable("artifact").Column("name", sql.Text()).PrimaryKey()
	require.NoError(t, db.Apply(up))
	require.ErrorIs(t, db.Apply(up), ErrDuplicate)

	other := sql.NewFile().Postgres()
	other.CreateTable("extra").Column("id", sql.BigInt()).PrimaryKey()
	err := db.Apply(other)
	require.ErrorIs(t, err, ErrInvalid)

	missing := sql.NewFile()
	missing.DropTable("absent")
	require.ErrorIs(t, db.Apply(missing), ErrNotFound)
	require.ErrorIs(t, db.Apply(missing), errs.ErrNotFound)

	var bare *Database
	_, err = bare.Make()
	require.ErrorIs(t, err, ErrInvalid)
}

func read(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Clean(rel))
	require.NoError(t, err)
	return string(body)
}

func describeDB(db *Database) string {
	var b strings.Builder
	b.WriteString(db.Dialect.String())
	b.WriteByte('\n')
	for _, table := range db.Tables {
		b.WriteString(table.Name)
		b.WriteString(" pk=")
		b.WriteString(strings.Join(table.Primary, ","))
		for _, col := range table.Columns {
			b.WriteByte(' ')
			b.WriteString(col.Name)
			b.WriteByte(':')
			b.WriteString(col.Type.SQL())
			if col.Primary {
				b.WriteString(":pk")
			}
			if col.Auto {
				b.WriteString(":auto")
			}
			if col.NotNull {
				b.WriteString(":nn")
			}
			if col.IsUnique {
				b.WriteString(":uq")
			}
			if col.Default != nil {
				b.WriteString(":default=")
				b.WriteString(col.Default.Kind.String())
				b.WriteByte(':')
				b.WriteString(col.Default.Text)
			}
		}
		for _, set := range table.Uniques {
			b.WriteString(" unique=")
			b.WriteString(strings.Join(set, ","))
		}
		for _, fk := range table.Foreign {
			b.WriteString(" fk=")
			b.WriteString(strings.Join(fk.Columns, ","))
			b.WriteByte('>')
			b.WriteString(fk.Table)
			b.WriteByte('(')
			b.WriteString(strings.Join(fk.RefColumns, ","))
			b.WriteByte(')')
			if fk.OnDelete.IsValid() {
				b.WriteByte(' ')
				b.WriteString(fk.OnDelete.SQL())
			}
			if fk.OnUpdate.IsValid() {
				b.WriteByte(' ')
				b.WriteString(fk.OnUpdate.SQL())
			}
		}
		for _, idx := range table.Indexes {
			b.WriteString(" index=")
			b.WriteString(idx.Name)
			b.WriteByte('(')
			b.WriteString(strings.Join(idx.Columns, ","))
			b.WriteByte(')')
			if idx.IsUnique {
				b.WriteString(":uq")
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
