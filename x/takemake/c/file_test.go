package c

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNilInputs(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, errRead)
	require.ErrorIs(t, (*File)(nil).Make(&bytes.Buffer{}), errFile)
	require.ErrorIs(t, New().Make(nil), errWrite)
}

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := New()
	file.IncludeSystem("stdio.h")
	file.IncludeQuoted("local.h")
	file.Const("ANSWER", 42)
	point := file.Struct("Point").
		Field("tag", Char()).
		Field("x", Int()).
		Field("y", Long()).
		Field("z", Double())
	require.Equal(t, 0, point.Fields[0].Offset)
	require.Equal(t, 4, point.Fields[1].Offset)
	require.Equal(t, 8, point.Fields[2].Offset)
	require.Equal(t, 16, point.Fields[3].Offset)
	require.Equal(t, 24, point.Size)
	require.Equal(t, 8, point.Align)
	use := point.Alias("Point")
	layout(use)
	require.Equal(t, 24, use.Size)
	require.Equal(t, 8, use.Align)

	node := file.Struct("Node")
	node.Field("next", Pointer(node)).Field("value", Int())
	require.Equal(t, 0, node.Fields[0].Offset)
	require.Equal(t, 8, node.Fields[1].Offset)
	require.Equal(t, 8, node.Fields[0].Type.Size)
	require.Equal(t, 16, node.Size)
	require.Equal(t, 8, node.Align)
	node.Alias("Node")

	slot := file.Union("Slot").
		Field("n", Int()).
		Field("p", Pointer(Void())).
		Field("d", Double())
	require.Equal(t, 8, slot.Size)
	require.Equal(t, 8, slot.Align)
	require.Equal(t, 0, slot.Fields[0].Offset)
	require.Equal(t, 0, slot.Fields[1].Offset)
	require.Equal(t, 0, slot.Fields[2].Offset)
	slot.Alias("Slot")

	color := file.Enum("Color").
		Enumerator("RED", 1).
		Enumerator("GREEN", 2).
		Enumerator("BLUE", 4)
	require.Equal(t, 4, color.Size)
	require.Equal(t, 4, color.Align)
	color.Alias("Color")

	wide := file.Enum("Wide").Enumerator("BIG", 1<<40)
	require.Equal(t, 8, wide.Size)
	require.Equal(t, 8, wide.Align)
	wide.Alias("Wide")

	file.Object("flag", Int().Constant()).WithInit(IntLit(1))
	file.Object("origin", use)
	file.Func("add").Returns(Int()).Param("a", Int()).Param("b", Int()).WithBody(
		Return(Binary(OpAdd, Ident("a"), Ident("b"))),
	)
	file.Func("proto").Returns(Int()).Param("a", Int()).Param("b", Int())

	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	first, printed := roundTrip(t, buf.String())
	got := declNamed(t, first, "Point")
	require.Equal(t, "Point", got.Name)
	require.True(t, got.Define)
	require.Equal(t, KindStruct, got.Type.Kind)
	require.Equal(t, 24, got.Type.Size)
	require.Equal(t, 8, got.Type.Align)
	require.Equal(t, []int{0, 4, 8, 16}, offsets(got.Type))

	nodeDecl := declNamed(t, first, "Node")
	require.Equal(t, 16, nodeDecl.Type.Size)
	require.Equal(t, 8, nodeDecl.Type.Align)
	require.Equal(t, []int{0, 8}, offsets(nodeDecl.Type))
	require.Equal(t, KindPointer, nodeDecl.Type.Fields[0].Type.Kind)
	require.Equal(t, 8, nodeDecl.Type.Fields[0].Type.Size)

	slotDecl := declNamed(t, first, "Slot")
	require.Equal(t, KindUnion, slotDecl.Type.Kind)
	require.Equal(t, 8, slotDecl.Type.Size)
	require.Equal(t, []int{0, 0, 0}, offsets(slotDecl.Type))

	colorDecl := declNamed(t, first, "Color")
	require.Equal(t, KindEnum, colorDecl.Type.Kind)
	require.Equal(t, 4, colorDecl.Type.Size)
	require.Equal(t, []int64{1, 2, 4}, enumValues(colorDecl.Type))

	wideDecl := declNamed(t, first, "Wide")
	require.Equal(t, 8, wideDecl.Type.Size)

	flag := objectNamed(t, first, "flag")
	require.True(t, flag.Type.Const)
	require.Equal(t, KindInteger, flag.Type.Kind)
	require.Equal(t, 4, flag.Type.Size)
	require.Equal(t, ExprInt, flag.Init.Kind)
	require.Equal(t, int64(1), flag.Init.Int)

	origin := objectNamed(t, first, "origin")
	require.Equal(t, "Point", origin.Type.Use)
	require.Equal(t, 24, origin.Type.Size)

	add := funcNamed(t, first, "add")
	require.NotNil(t, add.Body)
	require.Equal(t, StmtBlock, add.Body.Kind)
	require.Equal(t, StmtReturn, add.Body.Stmts[0].Kind)
	require.Equal(t, ExprBinary, add.Body.Stmts[0].Expr.Kind)
	require.Equal(t, OpAdd, add.Body.Stmts[0].Expr.Op)

	proto := funcNamed(t, first, "proto")
	require.Nil(t, proto.Body)
	require.Contains(t, printed, "int proto(int a, int b);")
}

func TestParseRoundTrip(t *testing.T) {
	t.Parallel()
	const src = "#include <stdio.h>\n" +
		"#include \"local.h\"\n" +
		"#define SUM (1 << 2)\n" +
		"#define EMPTY\n" +
		"typedef struct Point {\n" +
		"\tchar tag;\n" +
		"\tint x;\n" +
		"\tlong y;\n" +
		"\tdouble z;\n" +
		"} Point;\n" +
		"struct Node {\n" +
		"\tstruct Node *next;\n" +
		"\tint value;\n" +
		"};\n" +
		"union Slot {\n" +
		"\tint n;\n" +
		"\tvoid *p;\n" +
		"\tdouble d;\n" +
		"};\n" +
		"enum Color { RED, GREEN = 2, BLUE };\n" +
		"typedef int Number;\n" +
		"typedef int *Ptr;\n" +
		"const int flag = 1;\n" +
		"static Number count;\n" +
		"extern double scale;\n" +
		"int *cells[4];\n" +
		"int (*row)[4];\n" +
		"int tail[];\n" +
		"_Bool on;\n" +
		"int add(int a, int b);\n" +
		"int *idp(int);\n" +
		"int sum(int n) {\n" +
		"\tint total = 0;\n" +
		"\tif (n) {\n" +
		"\t\treturn n;\n" +
		"\t} else {\n" +
		"\t\ttotal = 0;\n" +
		"\t}\n" +
		"\twhile (n) {\n" +
		"\t\tn = count;\n" +
		"\t}\n" +
		"\tfor (total = 0; total < n; total = total + 1) {\n" +
		"\t\tbreak;\n" +
		"\t\tcontinue;\n" +
		"\t}\n" +
		"\tprintf(\"hi\");\n" +
		"\treturn total;\n" +
		"}\n" +
		"struct Node *walk(struct Node *n) {\n" +
		"\twhile (n) {\n" +
		"\t\tn = n->next;\n" +
		"\t}\n" +
		"\treturn n;\n" +
		"}\n"

	first, printed := roundTrip(t, src)
	require.Equal(t, "stdio.h", first.Decls[0].Include.Path)
	require.True(t, first.Decls[0].Include.System)
	require.Equal(t, "local.h", first.Decls[1].Include.Path)
	require.False(t, first.Decls[1].Include.System)
	require.Equal(t, int64(4), constNamed(t, first, "SUM").Value)
	require.Equal(t, int64(0), constNamed(t, first, "EMPTY").Value)
	require.Contains(t, printed, "#define SUM 4")

	point := declNamed(t, first, "Point")
	require.Equal(t, 24, point.Type.Size)
	require.Equal(t, 8, point.Type.Align)
	require.Equal(t, []int{0, 4, 8, 16}, offsets(point.Type))
	require.Equal(t, []string{"char", "int", "long", "double"}, fieldSpells(point.Type))

	node := typeTagged(t, first, "Node")
	require.Equal(t, 16, node.Size)
	require.Equal(t, 8, node.Align)
	require.Equal(t, 0, node.Fields[0].Offset)
	require.Equal(t, 8, node.Fields[1].Offset)
	require.Equal(t, KindPointer, node.Fields[0].Type.Kind)
	require.Equal(t, 8, node.Fields[0].Type.Size)

	slot := typeTagged(t, first, "Slot")
	require.Equal(t, KindUnion, slot.Kind)
	require.Equal(t, 8, slot.Size)
	require.Equal(t, 8, slot.Align)
	require.Equal(t, []int{0, 0, 0}, offsets(slot))

	color := typeTagged(t, first, "Color")
	require.Equal(t, KindEnum, color.Kind)
	require.Equal(t, 4, color.Size)
	require.Equal(t, []int64{0, 2, 3}, enumValues(color))
	require.Contains(t, printed, "GREEN = 2")

	require.Equal(t, "bool", objectNamed(t, first, "on").Type.Spell)
	require.Equal(t, 1, objectNamed(t, first, "on").Type.Size)

	cells := objectNamed(t, first, "cells")
	require.Equal(t, KindArray, cells.Type.Kind)
	require.Equal(t, 4, cells.Type.Len)
	require.Equal(t, 32, cells.Type.Size)
	require.Equal(t, KindPointer, cells.Type.Elem.Kind)

	row := objectNamed(t, first, "row")
	require.Equal(t, KindPointer, row.Type.Kind)
	require.Equal(t, 8, row.Type.Size)
	require.Equal(t, KindArray, row.Type.Elem.Kind)
	require.Equal(t, 4, row.Type.Elem.Len)
	require.Equal(t, 16, row.Type.Elem.Size)

	tail := objectNamed(t, first, "tail")
	require.Equal(t, -1, tail.Type.Len)
	require.Equal(t, 0, tail.Type.Size)

	ptr := declNamed(t, first, "Ptr")
	require.Equal(t, KindPointer, ptr.Type.Kind)
	require.Equal(t, 8, ptr.Type.Size)

	add := funcNamed(t, first, "add")
	require.Nil(t, add.Body)
	require.Contains(t, printed, "int add(int a, int b);")

	idp := funcNamed(t, first, "idp")
	require.Nil(t, idp.Body)
	require.Equal(t, KindPointer, idp.Result.Kind)
	require.Len(t, idp.Params, 1)
	require.Empty(t, idp.Params[0].Name)
	require.Equal(t, KindInteger, idp.Params[0].Type.Kind)

	sum := funcNamed(t, first, "sum")
	require.Equal(t, StmtBlock, sum.Body.Kind)
	require.Equal(t, []StmtKind{
		StmtDecl, StmtIf, StmtWhile, StmtFor, StmtExpr, StmtReturn,
	}, stmtKinds(sum.Body.Stmts))
	require.Equal(t, StmtReturn, sum.Body.Stmts[1].Body.Stmts[0].Kind)
	require.Equal(t, StmtExpr, sum.Body.Stmts[1].Else.Stmts[0].Kind)
	require.Equal(t, StmtExpr, sum.Body.Stmts[2].Body.Stmts[0].Kind)
	require.Equal(t, StmtExpr, sum.Body.Stmts[3].Init.Kind)
	require.Equal(t, ExprBinary, sum.Body.Stmts[3].Expr.Kind)
	require.Equal(t, []StmtKind{StmtBreak, StmtContinue}, stmtKinds(sum.Body.Stmts[3].Body.Stmts))
	require.Equal(t, ExprCall, sum.Body.Stmts[4].Expr.Kind)
	require.Zero(t, sum.Storage)

	count := objectNamed(t, first, "count")
	require.Equal(t, StorageStatic, count.Storage)
	require.Equal(t, "Number", count.Type.Use)
	require.Equal(t, StorageExtern, objectNamed(t, first, "scale").Storage)

	walk := funcNamed(t, first, "walk")
	assign := walk.Body.Stmts[0].Body.Stmts[0].Expr
	require.Equal(t, ExprAssign, assign.Kind)
	require.True(t, assign.Right.Arrow)
	require.Equal(t, "next", assign.Right.Text)
}

func TestNonTrivialRoundTrip(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("testdata/buffer.c")
	require.NoError(t, err)
	first, printed := roundTrip(t, string(src))
	require.NotContains(t, printed, "/*")
	require.NotContains(t, printed, "//")
	require.Contains(t, printed, "#define CAP 256")
	require.Contains(t, printed, "#define ALIGN 8")
	require.Contains(t, printed, "buf->data[buf->len]")
	require.Contains(t, printed, "doc->version.major")

	require.Equal(t, int64(256), constNamed(t, first, "CAP").Value)
	require.Equal(t, int64(8), constNamed(t, first, "ALIGN").Value)
	require.Equal(t, int64(0), constNamed(t, first, "EMPTY").Value)

	forward := typeDeclTag(t, first, "Buf", false)
	require.False(t, forward.Define)
	buf := declNamed(t, first, "Buf")
	require.True(t, buf.Define)
	require.Equal(t, 32, buf.Type.Size)
	require.Equal(t, 8, buf.Type.Align)
	require.Equal(t, []int{0, 8, 16, 24}, offsets(buf.Type))
	require.Equal(t, KindPointer, buf.Type.Fields[3].Type.Kind)
	require.Equal(t, 8, buf.Type.Fields[3].Type.Size)
	require.Equal(t, KindPointer, buf.Type.Fields[0].Type.Kind)
	require.False(t, buf.Type.Fields[0].Type.Elem.Const)
	require.Equal(t, "char", buf.Type.Fields[0].Type.Elem.Spell)

	span := declNamed(t, first, "Span")
	require.Equal(t, 16, span.Type.Size)
	require.Equal(t, 8, span.Type.Align)
	require.Equal(t, []int{0, 8}, offsets(span.Type))
	require.True(t, span.Type.Fields[0].Type.Elem.Const)
	require.Equal(t, "char", span.Type.Fields[0].Type.Elem.Spell)

	doc := declNamed(t, first, "Doc")
	require.Equal(t, 48, doc.Type.Size)
	require.Equal(t, 8, doc.Type.Align)
	require.Equal(t, []int{0, 16, 32}, offsets(doc.Type))
	version := doc.Type.Fields[1].Type
	require.Equal(t, KindStruct, version.Kind)
	require.True(t, version.Inline)
	require.Empty(t, version.Tag)
	require.Equal(t, 16, version.Size)
	require.Equal(t, []int{0, 8}, offsets(version))
	pages := doc.Type.Fields[2].Type
	require.Equal(t, KindArray, pages.Kind)
	require.Equal(t, 2, pages.Len)
	require.Equal(t, 16, pages.Size)
	require.Equal(t, KindPointer, pages.Elem.Kind)

	status := typeTagged(t, first, "Status")
	require.Equal(t, 4, status.Size)
	require.Equal(t, []int64{0, 1, 4}, enumValues(status))

	writeFn := declNamed(t, first, "WriteFn")
	require.Equal(t, KindPointer, writeFn.Type.Kind)
	require.Equal(t, 8, writeFn.Type.Size)
	require.Equal(t, KindFunction, writeFn.Type.Elem.Kind)
	require.Equal(t, KindInteger, writeFn.Type.Elem.Elem.Kind)
	require.Len(t, writeFn.Type.Elem.Params, 2)
	require.Equal(t, "ctx", writeFn.Type.Elem.Params[0].Name)
	require.Equal(t, "span", writeFn.Type.Elem.Params[1].Name)

	require.Equal(t, int64('x'), objectNamed(t, first, "mark").Init.Int)
	require.Equal(t, "hello\n", objectNamed(t, first, "greeting").Init.Text)
	require.Equal(t, 1.25, objectNamed(t, first, "ratio").Init.Float)
	require.Equal(t, int64(255), objectNamed(t, first, "mask").Init.Int)
	require.Equal(t, ExprSizeof, objectNamed(t, first, "span_size").Init.Kind)
	require.Equal(t, "Span", objectNamed(t, first, "span_size").Init.Type.Use)
	require.Equal(t, 256, objectNamed(t, first, "scratch").Type.Len)
	require.Equal(t, 256, objectNamed(t, first, "scratch").Type.Size)
	require.Equal(t, 4, objectNamed(t, first, "pad").Type.Len)
	table := objectNamed(t, first, "table")
	require.Equal(t, 4, table.Type.Len)
	require.Equal(t, 64, table.Type.Size)
	require.Equal(t, "Span", table.Type.Elem.Use)
	require.Equal(t, 16, table.Type.Elem.Size)
	ticks := objectNamed(t, first, "ticks")
	require.Equal(t, StorageExtern, ticks.Storage)
	require.True(t, ticks.Type.Volatile)
	require.Equal(t, int64(1), objectNamed(t, first, "room_a").Init.Int)
	require.Equal(t, int64(2), objectNamed(t, first, "room_b").Init.Int)

	logLine := funcNamed(t, first, "log_line")
	require.True(t, logLine.Variadic)
	require.Equal(t, StorageStatic, logLine.Storage)
	require.Equal(t, ExprCast, logLine.Body.Stmts[0].Expr.Kind)
	require.Equal(t, KindVoid, logLine.Body.Stmts[0].Expr.Type.Kind)

	writeSpan := funcNamed(t, first, "write_span")
	require.Equal(t, []StmtKind{
		StmtDecl, StmtDecl, StmtDecl, StmtExpr, StmtIf, StmtExpr, StmtIf, StmtExpr, StmtWhile, StmtReturn,
	}, stmtKinds(writeSpan.Body.Stmts))
	loop := writeSpan.Body.Stmts[8]
	require.Equal(t, ExprAssign, loop.Body.Stmts[0].Expr.Kind)
	require.Equal(t, ExprIndex, loop.Body.Stmts[0].Expr.Left.Kind)
	require.True(t, loop.Body.Stmts[0].Expr.Left.Left.Arrow)
	require.Equal(t, StmtContinue, loop.Body.Stmts[3].Body.Stmts[0].Kind)

	copyAll := funcNamed(t, first, "copy_all")
	require.Equal(t, []StmtKind{
		StmtDecl, StmtDecl, StmtExpr, StmtFor, StmtIf, StmtReturn,
	}, stmtKinds(copyAll.Body.Stmts))
	choice := copyAll.Body.Stmts[3].Body.Stmts[0]
	require.Equal(t, StmtIf, choice.Kind)
	require.Equal(t, StmtContinue, choice.Body.Stmts[0].Kind)
	require.Equal(t, StmtIf, choice.Else.Kind)
	require.Equal(t, StmtExpr, choice.Else.Else.Stmts[0].Kind)
	require.Equal(t, StmtBreak, copyAll.Body.Stmts[3].Body.Stmts[1].Body.Stmts[0].Kind)
	nested := copyAll.Body.Stmts[4].Body.Stmts[1]
	require.Equal(t, StmtBlock, nested.Kind)
	require.Equal(t, ExprAssign, nested.Stmts[0].Expr.Kind)
	require.Equal(t, ExprSizeof, nested.Stmts[0].Expr.Right.Kind)
	require.Equal(t, ExprField, nested.Stmts[0].Expr.Right.Left.Kind)
	require.Equal(t, ExprAssign, nested.Stmts[1].Expr.Kind)
	require.Equal(t, ExprCast, nested.Stmts[1].Expr.Right.Kind)

	get := funcNamed(t, first, "buf_get")
	require.Equal(t, KindPointer, get.Result.Kind)
	require.True(t, get.Result.Elem.Const)
	require.Equal(t, ExprUnary, get.Body.Stmts[1].Expr.Kind)
	require.Equal(t, OpAddr, get.Body.Stmts[1].Expr.Op)
	require.Equal(t, ExprIndex, get.Body.Stmts[1].Expr.Left.Kind)

	docOK := funcNamed(t, first, "doc_ok")
	require.True(t, docOK.Inline)
	require.Equal(t, StorageStatic, docOK.Storage)
	require.Equal(t, ExprBinary, docOK.Body.Stmts[1].Expr.Kind)
	require.Equal(t, OpAnd, docOK.Body.Stmts[1].Expr.Op)
	major := docOK.Body.Stmts[1].Expr.Left.Left
	require.Equal(t, "major", major.Text)
	require.True(t, major.Left.Arrow)
	require.Equal(t, "version", major.Left.Text)

	flag := funcNamed(t, first, "flag_on")
	require.Equal(t, ExprBinary, flag.Body.Stmts[0].Expr.Kind)
	require.Equal(t, OpOr, flag.Body.Stmts[0].Expr.Op)

	var bodies int
	for _, decl := range first.Decls {
		if decl.Kind == DeclFunc && decl.Func != nil && decl.Func.Body != nil {
			bodies++
		}
	}
	require.Equal(t, 8, bodies)
}

func TestCsmithRoundTrip(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir("testdata/csmith")
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".c") {
			continue
		}
		names = append(names, entry.Name())
	}
	require.NotEmpty(t, names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			src, err := os.ReadFile(filepath.Join("testdata/csmith", name))
			require.NoError(t, err)
			first, printed := roundTrip(t, string(src))
			require.NotContains(t, printed, "/*")
			var aggregates int
			for _, decl := range first.Decls {
				if decl.Kind != DeclType || decl.Type == nil || decl.Type.Type == nil {
					continue
				}
				switch decl.Type.Type.Kind {
				case KindStruct, KindUnion:
					aggregates++
				}
			}
			require.Positive(t, aggregates, printed)
		})
	}
}

func TestStressRoundTrip(t *testing.T) {
	t.Parallel()
	src, err := os.ReadFile("testdata/stress.c")
	require.NoError(t, err)
	first, printed := roundTrip(t, string(src))
	require.NotContains(t, printed, "/*")
	require.Equal(t, int64(3), constNamed(t, first, "N").Value)
	require.Equal(t, int64(13), constNamed(t, first, "M").Value)
	require.Equal(t, int64(4294967296), constNamed(t, first, "WIDE").Value)
	require.Equal(t, int64(-1), constNamed(t, first, "MASK").Value)
	require.Equal(t, int64(-8), constNamed(t, first, "NEG").Value)

	box := declNamed(t, first, "Box")
	require.Equal(t, 24, box.Type.Size)
	require.Equal(t, []int{0, 4, 8, 16}, offsets(box.Type))

	node := typeTagged(t, first, "Node")
	require.Equal(t, 64, node.Size)
	require.Equal(t, 8, node.Align)
	require.Equal(t, []int{0, 8, 16, 24, 32, 48, 64}, offsets(node))
	require.Equal(t, KindPointer, node.Fields[0].Type.Kind)
	require.Equal(t, node, node.Fields[0].Type.Elem)
	cmp := node.Fields[3].Type
	require.Equal(t, KindPointer, cmp.Kind)
	require.Equal(t, KindFunction, cmp.Elem.Kind)
	require.True(t, cmp.Elem.Params[0].Type.Elem.Const)
	require.True(t, node.Fields[4].Type.Inline)
	require.Equal(t, 16, node.Fields[4].Type.Size)
	require.Equal(t, -1, node.Fields[6].Type.Len)
	require.Equal(t, 0, node.Fields[6].Type.Size)

	view := typeTagged(t, first, "View")
	require.Equal(t, KindUnion, view.Kind)
	require.Equal(t, 16, view.Size)
	require.Equal(t, 16, view.Align)
	require.Equal(t, []int{0, 0, 0, 0, 0}, offsets(view))
	require.Equal(t, "long double", view.Fields[4].Type.Spell)

	rank := typeTagged(t, first, "Rank")
	require.Equal(t, 8, rank.Size)
	require.Equal(t, 8, rank.Align)
	require.Equal(t, []int64{1, 2, 1 << 30, 4294967296}, enumValues(rank))

	hold := typeTagged(t, first, "Hold")
	require.Equal(t, 16, hold.Size)
	require.Equal(t, []int{0, 4, 12}, offsets(hold))
	require.Equal(t, 8, hold.Fields[1].Type.Size)
	require.True(t, hold.Fields[1].Type.Inline)

	arr := typeTagged(t, first, "Arr")
	require.Equal(t, KindArray, arr.Fields[0].Type.Kind)
	require.Equal(t, KindPointer, arr.Fields[0].Type.Elem.Kind)
	require.Equal(t, KindPointer, arr.Fields[1].Type.Kind)
	require.Equal(t, KindArray, arr.Fields[1].Type.Elem.Kind)
	fnArr := arr.Fields[2].Type
	require.Equal(t, KindArray, fnArr.Kind)
	require.Equal(t, KindPointer, fnArr.Elem.Kind)
	require.Equal(t, KindFunction, fnArr.Elem.Elem.Kind)
	grid := arr.Fields[3].Type
	require.Equal(t, 2, grid.Len)
	require.Equal(t, 3, grid.Elem.Len)
	require.Equal(t, 4, grid.Elem.Elem.Len)
	require.Equal(t, 96, grid.Size)
	require.Equal(t, 144, arr.Size)

	mix := typeTagged(t, first, "Mix")
	require.Equal(t, 24, mix.Size)
	require.Equal(t, []int{0, 8, 16}, offsets(mix))
	require.Equal(t, KindPointer, mix.Fields[1].Type.Kind)
	require.Equal(t, KindArray, mix.Fields[2].Type.Kind)

	require.False(t, typeDeclTag(t, first, "Fwd", false).Define)
	handle := declNamed(t, first, "Handle")
	require.Equal(t, KindPointer, handle.Type.Kind)
	require.Equal(t, 8, handle.Type.Size)

	signal := funcNamed(t, first, "signal")
	require.Nil(t, signal.Body)
	require.Equal(t, KindPointer, signal.Result.Kind)
	require.Equal(t, KindFunction, signal.Result.Elem.Kind)
	require.Equal(t, KindVoid, signal.Result.Elem.Elem.Kind)
	require.Equal(t, KindPointer, signal.Params[1].Type.Kind)
	require.Equal(t, KindFunction, signal.Params[1].Type.Elem.Kind)

	wrap := funcNamed(t, first, "wrap")
	require.Equal(t, KindPointer, wrap.Result.Kind)
	require.Equal(t, KindFunction, wrap.Result.Elem.Kind)
	require.Equal(t, KindPointer, wrap.Result.Elem.Elem.Kind)
	require.Equal(t, KindArray, wrap.Result.Elem.Elem.Elem.Kind)
	require.Equal(t, 3, wrap.Result.Elem.Elem.Elem.Len)
	require.Equal(t, KindFunction, wrap.Params[0].Type.Elem.Kind)

	levels := objectNamed(t, first, "levels")
	require.True(t, levels.Type.Volatile)
	require.False(t, levels.Type.Const)
	require.True(t, levels.Type.Elem.Const)
	require.True(t, levels.Type.Elem.Elem.Const)
	require.Equal(t, "int", levels.Type.Elem.Elem.Spell)

	require.Equal(t, 16, objectNamed(t, first, "pi").Type.Size)
	require.Equal(t, 16, objectNamed(t, first, "pi").Type.Align)
	require.Equal(t, 3, objectNamed(t, first, "cells").Type.Len)
	require.Equal(t, 13, objectNamed(t, first, "row").Type.Elem.Len)
	require.Equal(t, -1, objectNamed(t, first, "tail").Type.Len)

	table := objectNamed(t, first, "table")
	require.Equal(t, 2, table.Type.Len)
	require.Equal(t, 3, table.Type.Elem.Len)
	require.Equal(t, KindPointer, table.Type.Elem.Elem.Kind)
	require.Equal(t, KindFunction, table.Type.Elem.Elem.Elem.Kind)
	require.Equal(t, 48, table.Type.Size)

	rowsFn := objectNamed(t, first, "rows_fn")
	require.Equal(t, KindPointer, rowsFn.Type.Kind)
	require.Equal(t, KindArray, rowsFn.Type.Elem.Kind)
	require.Equal(t, 2, rowsFn.Type.Elem.Len)
	require.Equal(t, KindPointer, rowsFn.Type.Elem.Elem.Kind)
	require.Equal(t, KindFunction, rowsFn.Type.Elem.Elem.Elem.Kind)

	deep := objectNamed(t, first, "deep")
	require.Equal(t, KindPointer, deep.Type.Kind)
	require.Equal(t, KindFunction, deep.Type.Elem.Kind)
	require.Equal(t, KindPointer, deep.Type.Elem.Elem.Kind)
	require.Equal(t, KindArray, deep.Type.Elem.Elem.Elem.Kind)
	require.Equal(t, 4, deep.Type.Elem.Elem.Elem.Len)
	require.Equal(t, KindPointer, deep.Type.Elem.Elem.Elem.Elem.Kind)
	require.Equal(t, KindFunction, deep.Type.Elem.Elem.Elem.Elem.Elem.Kind)

	fp := objectNamed(t, first, "fp")
	require.Equal(t, KindArray, fp.Type.Kind)
	require.Equal(t, 2, fp.Type.Len)
	require.Equal(t, KindPointer, fp.Type.Elem.Kind)
	require.Equal(t, KindFunction, fp.Type.Elem.Elem.Kind)
	require.True(t, fp.Type.Elem.Elem.Elem.Const)
	require.Equal(t, "long double", fp.Type.Elem.Elem.Elem.Elem.Spell)
	require.True(t, objectNamed(t, first, "variadic_cmp").Type.Elem.Variadic)

	one := objectNamed(t, first, "one")
	two := objectNamed(t, first, "two")
	require.Equal(t, node, one.Type.Elem)
	require.Equal(t, node, two.Type.Elem.Elem)
	require.Equal(t, node, objectNamed(t, first, "kids").Type.Elem.Elem)
	nest := declNamed(t, first, "Nest")
	require.Equal(t, KindPointer, nest.Type.Kind)
	require.Equal(t, KindFunction, nest.Type.Elem.Kind)
	require.Equal(t, KindPointer, nest.Type.Elem.Elem.Kind)
	require.Equal(t, KindArray, nest.Type.Elem.Elem.Elem.Kind)
	require.Equal(t, "Nest", objectNamed(t, first, "nest").Type.Use)

	require.Equal(t, "BinOp", objectNamed(t, first, "ops").Type.Elem.Use)
	require.Equal(t, 16, objectNamed(t, first, "ops").Type.Size)
	require.Equal(t, "abcd\n", objectNamed(t, first, "greeting").Init.Text)
	require.Equal(t, int64('a'), objectNamed(t, first, "mark").Init.Int)
	require.Equal(t, KindArray, declNamed(t, first, "Row").Type.Kind)
	require.Equal(t, KindPointer, declNamed(t, first, "Row").Type.Elem.Kind)

	call := funcNamed(t, first, "call_all")
	require.Equal(t, StorageStatic, call.Storage)
	require.Equal(t, []StmtKind{
		StmtDecl, StmtDecl, StmtDecl, StmtDecl, StmtDecl,
		StmtExpr, StmtIf, StmtFor, StmtFor, StmtWhile,
		StmtExpr, StmtExpr, StmtExpr, StmtExpr, StmtReturn,
	}, stmtKinds(call.Body.Stmts))
	blank := call.Body.Stmts[8]
	require.Nil(t, blank.Init)
	require.Nil(t, blank.Expr)
	require.Nil(t, blank.Post)
	require.Equal(t, ExprAssign, call.Body.Stmts[7].Post.Kind)
	wh := call.Body.Stmts[9].Expr
	require.Equal(t, OpAnd, wh.Op)
	require.Equal(t, ExprAssign, wh.Left.Left.Kind)
	require.Equal(t, "lo", wh.Left.Left.Right.Right.Text)
	require.True(t, wh.Left.Left.Right.Right.Left.Arrow)
	cast := call.Body.Stmts[10].Expr.Right
	require.Equal(t, ExprCast, cast.Kind)
	require.Equal(t, KindPointer, cast.Type.Kind)
	require.Equal(t, 4, cast.Type.Elem.Len)
	ret := call.Body.Stmts[14].Expr
	require.Equal(t, OpAdd, ret.Op)
	require.Equal(t, ExprCall, ret.Right.Kind)
	require.Equal(t, ExprSizeof, ret.Left.Left.Left.Kind)
	require.Equal(t, "Hold", ret.Left.Left.Left.Type.Tag)
	require.Equal(t, ExprUnary, ret.Left.Right.Left.Left.Kind)
	require.Equal(t, ExprCast, ret.Right.Args[0].Kind)
	require.Equal(t, KindPointer, ret.Right.Args[0].Type.Kind)
	require.Equal(t, KindFunction, ret.Right.Args[0].Type.Elem.Kind)
	require.Contains(t, printed, "void (*signal(int sig, void (*handler)(int)))(int);")
	require.Contains(t, printed, "int (*(*wrap(int (*fn)(int)))(void))[3];")
	require.Contains(t, printed, "int (*(*(*deep)(int))[4])(void);")

	walk := funcNamed(t, first, "walk")
	require.Equal(t, KindArray, walk.Params[0].Type.Kind)
	require.Equal(t, -1, walk.Params[0].Type.Len)
	require.Equal(t, 3, walk.Params[0].Type.Elem.Len)
	require.Equal(t, KindPointer, walk.Params[1].Type.Kind)
	require.Equal(t, KindFunction, walk.Params[1].Type.Elem.Kind)
	loop := walk.Body.Stmts[0]
	require.Equal(t, StmtFor, loop.Kind)
	require.Equal(t, StmtDecl, loop.Init.Kind)
	require.Equal(t, "i", loop.Init.Objects[0].Name)
	choice := loop.Body.Stmts[0]
	require.Equal(t, StmtIf, choice.Kind)
	require.Equal(t, StmtIf, choice.Else.Kind)
	require.Equal(t, StmtBreak, choice.Else.Else.Kind)
}

func TestQualifiedDeclarators(t *testing.T) {
	t.Parallel()
	const src = "int *const *p;\n" +
		"typedef const struct T { int x; } Name;\n" +
		"const struct T *q;\n" +
		"int (*fp)(int);\n"
	first, printed := roundTrip(t, src)
	p := objectNamed(t, first, "p")
	require.Equal(t, KindPointer, p.Type.Kind)
	require.False(t, p.Type.Const)
	require.True(t, p.Type.Elem.Const)
	require.Equal(t, KindPointer, p.Type.Elem.Kind)
	require.Contains(t, printed, "int *const *p;")

	name := declNamed(t, first, "Name")
	require.True(t, name.Define)
	require.True(t, name.Type.Const)
	require.Equal(t, KindStruct, name.Type.Of.Kind)
	require.Equal(t, 4, name.Type.Size)
	require.Equal(t, 0, name.Type.Of.Fields[0].Offset)
	require.Contains(t, printed, "typedef const struct T {")

	q := objectNamed(t, first, "q")
	require.Equal(t, KindPointer, q.Type.Kind)
	require.True(t, q.Type.Elem.Const)
	require.Equal(t, "T", q.Type.Elem.Tag)

	fp := objectNamed(t, first, "fp")
	require.Equal(t, KindPointer, fp.Type.Kind)
	require.Equal(t, KindFunction, fp.Type.Elem.Kind)
	require.Equal(t, 8, fp.Type.Size)
}

func TestErrors(t *testing.T) {
	t.Parallel()
	_, err := Take(strings.NewReader("int ;\n"))
	require.ErrorIs(t, err, errSyntax)

	_, err = Take(strings.NewReader("#define ADD(a, b) ((a)+(b))\n"))
	require.ErrorIs(t, err, errForm)

	_, err = Take(strings.NewReader("#if 0\nint x;\n#endif\n"))
	require.ErrorIs(t, err, errForm)

	_, err = Take(strings.NewReader("struct S { int x : 1; };\n"))
	require.ErrorIs(t, err, errForm)

	_, err = Take(strings.NewReader("void f(void) { x += 1; }\n"))
	require.ErrorIs(t, err, errForm)

	_, err = Take(strings.NewReader("void f(void) { switch (x) { } }\n"))
	require.ErrorIs(t, err, errForm)

	_, err = Take(strings.NewReader("struct T { int x; };\nstruct T { int y; };\n"))
	require.ErrorIs(t, err, errSyntax)
}

func roundTrip(t *testing.T, src string) (*File, string) {
	t.Helper()
	first, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	second, err := Take(strings.NewReader(buf.String()))
	require.NoError(t, err)
	require.Equal(t, dump(first), dump(second), buf.String())
	var again bytes.Buffer
	require.NoError(t, second.Make(&again))
	require.Equal(t, buf.String(), again.String())
	return first, buf.String()
}

func declNamed(t *testing.T, f *File, name string) *TypeDecl {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclType && d.Type != nil && d.Type.Name == name {
			return d.Type
		}
	}
	t.Fatalf("missing typedef %s\n%s", name, dump(f))
	return nil
}

func typeDeclTag(t *testing.T, f *File, tag string, define bool) *TypeDecl {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclType && d.Type != nil && d.Type.Name == "" && d.Type.Define == define && d.Type.Type != nil && d.Type.Type.Tag == tag {
			return d.Type
		}
	}
	t.Fatalf("missing %s define=%v\n%s", tag, define, dump(f))
	return nil
}

func typeTagged(t *testing.T, f *File, tag string) *Type {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclType && d.Type != nil && d.Type.Type != nil && d.Type.Type.Tag == tag && d.Type.Define {
			return d.Type.Type
		}
	}
	t.Fatalf("missing tag %s\n%s", tag, dump(f))
	return nil
}

func objectNamed(t *testing.T, f *File, name string) *Object {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclObject && d.Object != nil && d.Object.Name == name {
			return d.Object
		}
	}
	t.Fatalf("missing object %s\n%s", name, dump(f))
	return nil
}

func funcNamed(t *testing.T, f *File, name string) *Func {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclFunc && d.Func != nil && d.Func.Name == name {
			return d.Func
		}
	}
	t.Fatalf("missing func %s\n%s", name, dump(f))
	return nil
}

func constNamed(t *testing.T, f *File, name string) *Const {
	t.Helper()
	for _, d := range f.Decls {
		if d.Kind == DeclConst && d.Const != nil && d.Const.Name == name {
			return d.Const
		}
	}
	t.Fatalf("missing const %s\n%s", name, dump(f))
	return nil
}

func offsets(t *Type) []int {
	out := make([]int, len(t.Fields))
	for i, field := range t.Fields {
		out[i] = field.Offset
	}
	return out
}

func enumValues(t *Type) []int64 {
	out := make([]int64, len(t.Enums))
	for i, item := range t.Enums {
		out[i] = item.Value
	}
	return out
}

func fieldSpells(t *Type) []string {
	out := make([]string, len(t.Fields))
	for i, field := range t.Fields {
		out[i] = field.Type.Spell
	}
	return out
}

func stmtKinds(list []*Stmt) []StmtKind {
	out := make([]StmtKind, len(list))
	for i, s := range list {
		out[i] = s.Kind
	}
	return out
}

func dump(f *File) string {
	var b strings.Builder
	seen := map[*Type]int{}
	next := 1
	for _, d := range f.Decls {
		if d == nil {
			b.WriteString("nil\n")
			continue
		}
		b.WriteString(d.Kind.String())
		b.WriteByte(' ')
		switch d.Kind {
		case DeclInclude:
			if d.Include != nil {
				b.WriteString(d.Include.Path)
				if d.Include.System {
					b.WriteString(" system")
				}
			}
		case DeclConst:
			if d.Const != nil {
				b.WriteString(d.Const.Name)
				b.WriteByte('=')
				b.WriteString(itoa(d.Const.Value))
			}
		case DeclType:
			writeTypeDecl(&b, d.Type, seen, &next)
		case DeclFunc:
			writeFunc(&b, d.Func, seen, &next)
		case DeclObject:
			writeObject(&b, d.Object, seen, &next)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func writeTypeDecl(b *strings.Builder, d *TypeDecl, seen map[*Type]int, next *int) {
	if d == nil {
		return
	}
	b.WriteString(d.Name)
	if d.Define {
		b.WriteString(" define")
	}
	b.WriteByte(' ')
	writeType(b, d.Type, seen, next)
}

func writeFunc(b *strings.Builder, fn *Func, seen map[*Type]int, next *int) {
	if fn == nil {
		return
	}
	b.WriteString(fn.Name)
	b.WriteByte(' ')
	b.WriteString(fn.Storage.String())
	if fn.Inline {
		b.WriteString(" inline")
	}
	if fn.Variadic {
		b.WriteString(" variadic")
	}
	b.WriteString(" -> ")
	writeType(b, fn.Result, seen, next)
	b.WriteString(" (")
	for i, param := range fn.Params {
		if i > 0 {
			b.WriteString(", ")
		}
		if param == nil {
			continue
		}
		b.WriteString(param.Name)
		b.WriteByte(':')
		writeType(b, param.Type, seen, next)
	}
	b.WriteString(") ")
	writeStmt(b, fn.Body, seen, next)
}

func writeObject(b *strings.Builder, obj *Object, seen map[*Type]int, next *int) {
	if obj == nil {
		return
	}
	b.WriteString(obj.Name)
	b.WriteByte(' ')
	b.WriteString(obj.Storage.String())
	b.WriteByte(' ')
	writeType(b, obj.Type, seen, next)
	if obj.Init != nil {
		b.WriteString(" = ")
		writeExpr(b, obj.Init, seen, next)
	}
}

func writeType(b *strings.Builder, t *Type, seen map[*Type]int, next *int) {
	if t == nil {
		b.WriteString("nil")
		return
	}
	if id, ok := seen[t]; ok {
		b.WriteByte('#')
		b.WriteString(itoa(int64(id)))
		return
	}
	id := *next
	*next++
	seen[t] = id
	b.WriteByte('#')
	b.WriteString(itoa(int64(id)))
	b.WriteByte('{')
	b.WriteString(t.Kind.String())
	b.WriteString(" s=")
	b.WriteString(itoa(int64(t.Size)))
	b.WriteString(" a=")
	b.WriteString(itoa(int64(t.Align)))
	if t.Signed {
		b.WriteString(" signed")
	}
	if t.Const {
		b.WriteString(" const")
	}
	if t.Volatile {
		b.WriteString(" volatile")
	}
	if t.Spell != "" {
		b.WriteString(" spell=")
		b.WriteString(t.Spell)
	}
	if t.Use != "" {
		b.WriteString(" use=")
		b.WriteString(t.Use)
	}
	if t.Tag != "" {
		b.WriteString(" tag=")
		b.WriteString(t.Tag)
	}
	if t.Kind == KindArray {
		b.WriteString(" len=")
		b.WriteString(itoa(int64(t.Len)))
	}
	if t.Variadic {
		b.WriteString(" variadic")
	}
	if t.Of != nil {
		b.WriteString(" of=")
		writeType(b, t.Of, seen, next)
	}
	if t.Elem != nil {
		b.WriteString(" elem=")
		writeType(b, t.Elem, seen, next)
	}
	for _, field := range t.Fields {
		b.WriteString(" field ")
		if field == nil {
			b.WriteString("nil")
			continue
		}
		b.WriteString(field.Name)
		b.WriteByte('@')
		b.WriteString(itoa(int64(field.Offset)))
		b.WriteByte(':')
		writeType(b, field.Type, seen, next)
	}
	for _, item := range t.Enums {
		b.WriteString(" enum ")
		if item == nil {
			b.WriteString("nil")
			continue
		}
		b.WriteString(item.Name)
		b.WriteByte('=')
		b.WriteString(itoa(item.Value))
	}
	for _, param := range t.Params {
		b.WriteString(" param ")
		if param == nil {
			b.WriteString("nil")
			continue
		}
		b.WriteString(param.Name)
		b.WriteByte(':')
		writeType(b, param.Type, seen, next)
	}
	b.WriteByte('}')
}

func writeStmt(b *strings.Builder, s *Stmt, seen map[*Type]int, next *int) {
	if s == nil {
		b.WriteString("nil")
		return
	}
	b.WriteString(s.Kind.String())
	b.WriteByte('(')
	if s.Expr != nil {
		writeExpr(b, s.Expr, seen, next)
	}
	if s.Init != nil {
		b.WriteString(" init=")
		writeStmt(b, s.Init, seen, next)
	}
	if s.Post != nil {
		b.WriteString(" post=")
		writeExpr(b, s.Post, seen, next)
	}
	if s.Body != nil {
		b.WriteString(" body=")
		writeStmt(b, s.Body, seen, next)
	}
	if s.Else != nil {
		b.WriteString(" else=")
		writeStmt(b, s.Else, seen, next)
	}
	for _, child := range s.Stmts {
		b.WriteByte(' ')
		writeStmt(b, child, seen, next)
	}
	for _, obj := range s.Objects {
		b.WriteString(" obj=")
		writeObject(b, obj, seen, next)
	}
	b.WriteByte(')')
}

func writeExpr(b *strings.Builder, e *Expr, seen map[*Type]int, next *int) {
	if e == nil {
		b.WriteString("nil")
		return
	}
	b.WriteString(e.Kind.String())
	if e.Op != 0 {
		b.WriteByte(':')
		b.WriteString(e.Op.String())
	}
	if e.Text != "" {
		b.WriteByte(' ')
		b.WriteString(e.Text)
	}
	if e.Kind == ExprInt || e.Kind == ExprChar {
		b.WriteByte(' ')
		b.WriteString(itoa(e.Int))
	}
	if e.Kind == ExprFloat {
		b.WriteByte(' ')
		b.WriteString(strings.TrimRight(strings.TrimRight(formatFloat(e.Float), "0"), "."))
	}
	if e.Arrow {
		b.WriteString(" arrow")
	}
	if e.Type != nil {
		b.WriteString(" type=")
		writeType(b, e.Type, seen, next)
	}
	if e.Left != nil {
		b.WriteString(" left=")
		writeExpr(b, e.Left, seen, next)
	}
	if e.Right != nil {
		b.WriteString(" right=")
		writeExpr(b, e.Right, seen, next)
	}
	for _, arg := range e.Args {
		b.WriteString(" arg=")
		writeExpr(b, arg, seen, next)
	}
}

func itoa(v int64) string {
	return strconv.FormatInt(v, 10)
}
