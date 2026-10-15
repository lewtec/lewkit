package llvm

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
	point := Struct(IntType(32), IntType(32))
	file := New().Source("add.c").Layout("e-m:e-i64:64").Triple("x86_64-pc-linux-gnu")
	file.Type("point").Struct(IntType(32), IntType(32))
	file.Type("handle").Opaque()
	file.Global("g").SetDSOLocal().Type(IntType(32)).Init(Int(32, "42")).Align(4)
	file.Global("msg").Private().UnnamedAddr().Constant().Type(Array(6, IntType(8))).Init(CString("hi\n\x00")).Align(1)
	file.Global("ext").External().Type(IntType(32))
	file.Global("p").Type(Ptr()).Init(ConstGEP(
		Array(6, IntType(8)),
		Ptr(),
		GlobalRef("msg"),
		Typed(IntType(64), Int(64, "0")),
		Typed(IntType(64), Int(64, "0")),
	).Inbounds())
	file.Func("puts").Returns(IntType(32)).Param("", Ptr())
	file.Func("printf").SetDSOLocal().Returns(IntType(32)).Param("", Ptr()).VarArg()
	fn := file.Func("add").Private().SetDSOLocal().Returns(IntType(32)).Param("a", IntType(32)).Param("b", IntType(32))
	entry := fn.Block("entry")
	entry.Bin("sum", OpAdd, IntType(32), Local("a"), Local("b")).NSW().NUW()
	entry.ICmp("c", IPredSLT, IntType(32), Local("sum"), Int(32, "0"))
	entry.FCmp("ok", FPredOEQ, FP(FloatFloat), FloatLit(FloatFloat, "1.0"), FloatLit(FloatFloat, "2.0"))
	entry.Cast("wide", CastSExt, IntType(32), Local("sum"), IntType(64))
	entry.Alloca("slot", IntType(32)).Align(4)
	entry.Store(IntType(32), Local("sum"), Ptr(), Local("slot")).Volatile().Align(4)
	entry.Load("v", IntType(32), Ptr(), Local("slot")).Volatile().Align(4)
	entry.GEP("q", Array(4, IntType(32)), Ptr(), Local("arr"), Typed(IntType(64), Int(64, "0"))).Inbounds()
	entry.Extract("f0", point, Local("agg"), 0)
	entry.Insert("f1", point, Local("agg"), Typed(IntType(32), Int(32, "1")), 1)
	entry.Call("n", IntType(32), GlobalRef("puts"), Typed(Ptr(), Local("slot")))
	entry.CallSig("w", FuncType(IntType(32), Ptr()).VarArg(), GlobalRef("printf"), Typed(Ptr(), GlobalRef("msg"))).MustTail()
	entry.Phi("x", IntType(32), IncomingFrom(Int(32, "1"), "entry"), IncomingFrom(Local("v"), "loop"))
	entry.Bin("q", OpSDiv, IntType(32), Int(32, "8"), Int(32, "2")).Exact()
	entry.CondBr(Local("c"), "then", "exit")
	fn.Block("then").Br("entry")
	exit := fn.Block("exit")
	exit.Ret(IntType(32), Local("sum"))
	fn.Block("dead").Unreachable()
	file.Func("a b").Returns(Void()).Block("0").RetVoid()
	file.Func("space").Returns(Ptr().Space(1)).Block("entry").Ret(Ptr().Space(1), Null(Ptr().Space(1)))

	got, text := roundTripFile(t, file)
	require.Equal(t, file, got)
	require.Contains(t, text, `source_filename = "add.c"`)
	require.Contains(t, text, "define private dso_local i32 @add(i32 %a, i32 %b)")
	require.Contains(t, text, "%sum = add nuw nsw i32 %a, %b")
	require.Contains(t, text, `@"a b"`)
	require.Contains(t, text, "ptr addrspace(1)")
	require.NotContains(t, text, "noundef")
}

func TestSubsetRoundTrip(t *testing.T) {
	t.Parallel()
	const src = `
source_filename = "a.c"
target datalayout = "e-m:e-i64:64"
target triple = "x86_64-pc-linux-gnu"

%point = type { i32, i32 }
%handle = type opaque
%packed = type <{ i32, i8 }>

@g = dso_local global i32 42, align 4
@s = private unnamed_addr constant [2 x i8] c"a\00", align 1
@ext = external global i32
@agg = global { i32, i32 } { i32 1, i32 2 }
@vec = global <2 x i32> < i32 1, i32 2 >
@pk = global <{ i32, i8 }> <{ i32 1, i8 2 }>
@z = global { i32, i32 } zeroinitializer
@p = global ptr getelementptr inbounds ([2 x i8], ptr @s, i64 0, i64 0)
@tp = global i32 addrspace(1)* getelementptr inbounds ([2 x i8], [2 x i8] addrspace(1)* @s, i64 0, i64 0)
@named = global %point { i32 1, i32 2 }

declare dso_local i32 @printf(i8*, ...)

define internal i32 @use(i32 %a, i8* %p) {
entry:
	%slot = alloca i32, i32 1, align 4
	store i32 %a, i8* %p, align 4
	%v = load i32, i8* %p, align 4
	%q = getelementptr inbounds [4 x i32], [4 x i32]* %arr, i64 0, i64 1
	%bit = bitcast i8* %p to i32*
	%c = icmp eq i32 %v, 0
	%d = fcmp uno double 0.0, 1.0
	%f0 = extractvalue %point %agg, 0, 1
	%f1 = insertvalue %point %agg, i32 7, 1
	%x = phi i32 [ 1, %entry ], [ %v, %loop ]
	br i1 %c, label %then, label %exit
then:
	br label %exit
exit:
	%n = tail call i32 (i8*, ...) @printf(i8* %p)
	ret i32 %v
dead:
	unreachable
}
`
	first, text := roundTripSource(t, src)
	require.Equal(t, "a.c", first.SourceFilename)
	require.Equal(t, "e-m:e-i64:64", first.DataLayout)
	require.Len(t, first.Types, 3)
	require.True(t, first.Types[1].IsOpaque)
	require.Equal(t, KindPtr, first.Globals[8].ValueType.Kind)
	require.NotNil(t, first.Globals[8].ValueType.Elem)
	require.Equal(t, 1, first.Globals[8].ValueType.AddrSpace)
	require.Contains(t, text, "i32 addrspace(1)*")
	require.Contains(t, text, "unreachable")
}

func TestHelloOpaque(t *testing.T) {
	t.Parallel()
	first, text := roundTripSource(t, readTestdata(t, "c_hello_22.ll"))
	require.Equal(t, "testdata/ir/c_hello/source.c", first.SourceFilename)
	require.Equal(t, "main", first.Funcs[0].Name)
	require.True(t, first.Funcs[0].Defined)
	require.Equal(t, InstAlloca, first.Funcs[0].Blocks[0].Instructions[0].Kind)
	require.Equal(t, KindPtr, first.Funcs[0].Blocks[0].Instructions[2].Signature.Fields[0].Kind)
	require.Nil(t, first.Funcs[0].Blocks[0].Instructions[2].Signature.Fields[0].Elem)
	require.NotContains(t, text, "noundef")
	require.NotContains(t, text, "attributes")
	require.NotContains(t, text, "!llvm")
	require.Contains(t, text, "call i32 (ptr, ...) @printf(ptr @.str, ptr @.str.1)")
}

func TestHelloTyped(t *testing.T) {
	t.Parallel()
	first, text := roundTripSource(t, readTestdata(t, "c_hello_14.ll"))
	call := first.Funcs[0].Blocks[0].Instructions[2]
	require.Equal(t, KindFunc, call.Signature.Kind)
	require.True(t, call.Signature.Variadic)
	require.NotNil(t, call.Args[0].Value)
	require.Equal(t, ValGEP, call.Args[0].Value.Kind)
	require.True(t, call.Args[0].Value.InBounds)
	require.NotContains(t, text, "noundef")
	require.Contains(t, text, "getelementptr inbounds ([11 x i8], [11 x i8]* @.str, i64 0, i64 0)")
	require.Contains(t, text, "i8*")
	require.NotContains(t, text, " ptr ")
}

func TestCommentsAndRejects(t *testing.T) {
	t.Parallel()
	const src = `
; a comment
source_filename = "c.c"
define void @f() {
entry:
	; inside
	ret void
}
`
	file, _ := roundTripSource(t, src)
	require.Equal(t, "c.c", file.SourceFilename)
	require.Equal(t, InstRet, file.Funcs[0].Blocks[0].Instructions[0].Kind)

	_, err := Take(strings.NewReader("define void @f() {\nentry:\n\tfreeze i32 0\n\tret void\n}\n"))
	require.ErrorIs(t, err, ErrUnsupported)
	require.ErrorIs(t, err, errs.ErrUnsupported)

	_, err = Take(strings.NewReader("module asm \"nop\"\n"))
	require.ErrorIs(t, err, ErrUnsupported)

	_, err = Take(strings.NewReader("define void @f() {\nentry:\n\t%a = add fast i32 1, 2\n\tret void\n}\n"))
	require.ErrorIs(t, err, ErrUnsupported)

	_, err = Take(strings.NewReader("???\n"))
	require.ErrorIs(t, err, ErrSyntax)
	require.ErrorIs(t, err, errs.ErrSyntax)
}

func TestNilAndInvalid(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)

	err = (*File)(nil).Make(&bytes.Buffer{})
	require.ErrorIs(t, err, ErrNilFile)

	err = New().Make(nil)
	require.ErrorIs(t, err, ErrNilWriter)

	broken := New()
	broken.Func("f")
	err = broken.Make(&bytes.Buffer{})
	require.ErrorIs(t, err, ErrModule)

	bad := New()
	bad.Func("f").Returns(Void()).Link(Linkage("nope"))
	err = bad.Make(&bytes.Buffer{})
	require.ErrorIs(t, err, ErrLinkage)
	require.ErrorIs(t, err, errs.ErrUnknown)
}

func roundTripSource(t *testing.T, src string) (*File, string) {
	t.Helper()
	first, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	text := makeOf(t, first)
	second, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, text, makeOf(t, second))
	return first, text
}

func roundTripFile(t *testing.T, file *File) (*File, string) {
	t.Helper()
	text := makeOf(t, file)
	got, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, text, makeOf(t, got))
	return got, text
}

func makeOf(t *testing.T, file *File) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return buf.String()
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return string(raw)
}
