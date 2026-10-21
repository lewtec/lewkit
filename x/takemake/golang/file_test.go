package golang

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestTypeComment(t *testing.T) {
	t.Parallel()
	file := NewFile("lucide")
	file.Type().Id("Name").Of(Named("string")).Comment("Name is one icon.\n//\n//lewkit:enum")
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	text := buf.String()
	require.Contains(t, text, "// Name is one icon.\n//\n//lewkit:enum\ntype Name string")
}

func TestChain(t *testing.T) {
	t.Parallel()
	file := NewFile("main")
	fn := file.Func().Id("main").Params().Block(
		Qual("fmt", "Println").Call(Lit("hi")),
	)
	require.Equal(t, "main", file.Name)
	require.Equal(t, fn, file.Decls[0])
	require.Equal(t, "main", fn.Name)
	require.Empty(t, fn.Parameters)
	require.Empty(t, fn.Returns)
	require.True(t, fn.HasBody)
	require.Len(t, fn.Body, 1)
	call := fn.Body[0].Expr
	require.Equal(t, ExprCall, call.Kind)
	require.Equal(t, ExprSelector, call.Fun.Kind)
	require.Equal(t, "fmt", call.Fun.Path)
	require.Equal(t, "Println", call.Fun.Name)
	require.Equal(t, LitString, call.Args[0].Lit)
	require.Equal(t, `"hi"`, call.Args[0].Text)
	require.Equal(t, "fmt", file.Imports[0].Path)

	first := roundTripFile(t, file)
	got := first.Decls[0].(*Func)
	require.Equal(t, "main", got.Name)
	require.Equal(t, "fmt", got.Body[0].Expr.Fun.Path)
	require.Equal(t, "Println", got.Body[0].Expr.Fun.Name)
	require.Equal(t, `"hi"`, got.Body[0].Expr.Args[0].Text)
}

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := NewFile("sample")
	file.Type().Id("User").Struct(
		NewField(Named("string"), "Name").WithTag(`json:"name"`),
		NewField(Pointer(Named("int")), "Age"),
	)
	file.Const().Id("answer").Values(Binary(OpAdd, Lit(1), Binary(OpMul, Lit(2), Lit(3))))
	file.Var().Id("items").Type(Slice(Named("string")))
	file.Func().Recv("u", Pointer(Named("User"))).Id("String").Params().Results(NewField(Named("string"))).Block(
		Return(Id("u").Sel("Name")),
	)
	file.Func().Id("main").Params().Block(
		Qual("fmt", "Println").Call(Lit("hi")),
		Define([]*Expr{Id("user")}, Composite(Named("User"), KeyValue(Id("Name"), Lit("ada")))),
		Assign(0, []*Expr{Id("items")}, Composite(Slice(Named("string")), Elem(Lit("a")), Elem(Lit("b")))),
	)

	first := roundTripFile(t, file)
	require.Equal(t, "sample", first.Name)
	require.Equal(t, "fmt", first.Imports[0].Path)
	user := first.Decls[0].(*TypeDecl)
	require.Equal(t, "User", user.Specs[0].Name)
	require.Equal(t, TypeStruct, user.Specs[0].Type.Kind)
	require.Equal(t, `json:"name"`, user.Specs[0].Type.Fields[0].Tag)
	require.Equal(t, TypePointer, user.Specs[0].Type.Fields[1].Type.Kind)
	answer := first.Decls[1].(*Const)
	require.Equal(t, "answer", answer.Specs[0].Names[0])
	require.Equal(t, ExprBinary, answer.Specs[0].Values[0].Kind)
	require.Equal(t, OpAdd, answer.Specs[0].Values[0].Op)
	require.Equal(t, OpMul, answer.Specs[0].Values[0].Y.Op)
	method := first.Decls[3].(*Func)
	require.Equal(t, "String", method.Name)
	require.Equal(t, "u", method.Receiver.Names[0])
	require.Equal(t, "Name", method.Body[0].Exprs[0].Name)
}

func TestSourceRoundTrip(t *testing.T) {
	t.Parallel()
	const src = `package sample

import (
	"fmt"
	other "strconv"
)

const (
	answer = 1 + 2*3
	label  = "hi"
	raw    = ` + "`back`" + `
	runeC  = 'a'
	flt    = 1.5
	flag   = true
	groupA = iota
	groupB
)

var (
	count *int
	items []string
	table map[string]int
	buf   [4]byte
	fn    func(string) (int, error)
	box   struct {
		Name string ` + "`json:\"name\"`" + `
		Age  *int
	}
	handler interface {
		Serve(string) error
	}
	ch chan int
	ro <-chan int
	so chan<- int
)

type User struct {
	Name string ` + "`json:\"name\"`" + `
	Age  *int
	Tags []string
}

type Handler interface {
	Serve(name string, extra ...int) (int, error)
	fmt.Stringer
}

type Transform func(string) string

type Alias = User

func main() {
	fmt.Println("hi")
	n, err := other.Atoi("-12")
	if err != nil {
		return
	}
	items = []string{"a", "b"}
	user := User{Name: "ada", Age: nil}
	_ = -answer
	_ = n * 2
	_ = &user
	_ = user.Name
	_ = items[0]
	_ = buf[1:3]
	_ = flag
	_ = map[string]int{"a": 1}
	if n > 0 {
		n++
	} else {
		n--
	}
	for i := 0; i < n; i++ {
		_ = i
	}
	for k, v := range table {
		_ = k
		_ = v
	}
	for range items {
		break
	}
	defer fmt.Println("bye")
	go fmt.Println("side")
	const local = 1
	var tmp int
	_ = local
	_ = tmp
	{
		_ = raw
	}
	_ = func(z int) int { return z }(n)
}

func (u *User) String() string {
	return u.Name
}

func add(a, b int) int {
	return a + b
}

func echo(args ...string) {
	fmt.Println(args...)
}
`
	file := roundTrip(t, src)
	require.Equal(t, "sample", file.Name)
	require.Equal(t, "fmt", file.Imports[0].Path)
	require.Equal(t, "", file.Imports[0].Name)
	require.Equal(t, "strconv", file.Imports[1].Path)
	require.Equal(t, "other", file.Imports[1].Name)

	group := file.Decls[0].(*Const)
	require.Equal(t, "answer", group.Specs[0].Names[0])
	require.Equal(t, OpAdd, group.Specs[0].Values[0].Op)
	require.Equal(t, OpMul, group.Specs[0].Values[0].Y.Op)
	require.Equal(t, LitString, group.Specs[1].Values[0].Lit)
	require.Nil(t, group.Specs[7].Values)

	vars := file.Decls[1].(*Var)
	require.Equal(t, TypePointer, vars.Specs[0].Type.Kind)
	require.Equal(t, TypeSlice, vars.Specs[1].Type.Kind)
	require.Equal(t, TypeMap, vars.Specs[2].Type.Kind)
	require.Equal(t, TypeArray, vars.Specs[3].Type.Kind)
	require.Equal(t, LitInt, vars.Specs[3].Type.Len.Lit)
	require.Equal(t, "4", vars.Specs[3].Type.Len.Text)
	require.Equal(t, TypeFunc, vars.Specs[4].Type.Kind)
	require.Equal(t, TypeStruct, vars.Specs[5].Type.Kind)
	require.Equal(t, `json:"name"`, vars.Specs[5].Type.Fields[0].Tag)
	require.Equal(t, TypeInterface, vars.Specs[6].Type.Kind)
	require.Equal(t, TypeChan, vars.Specs[7].Type.Kind)
	require.Equal(t, ChanBoth, vars.Specs[7].Type.Dir)
	require.Equal(t, ChanRecv, vars.Specs[8].Type.Dir)
	require.Equal(t, ChanSend, vars.Specs[9].Type.Dir)

	user := file.Decls[2].(*TypeDecl)
	require.Equal(t, "User", user.Specs[0].Name)
	handler := file.Decls[3].(*TypeDecl)
	require.Equal(t, "Handler", handler.Specs[0].Name)
	serve := handler.Specs[0].Type.Fields[0]
	require.Equal(t, "Serve", serve.Names[0])
	require.True(t, serve.Type.Fields[1].Variadic)
	embedded := handler.Specs[0].Type.Fields[1]
	require.Empty(t, embedded.Names)
	require.Equal(t, "fmt", embedded.Type.Path)
	require.Equal(t, "Stringer", embedded.Type.Name)
	alias := file.Decls[5].(*TypeDecl)
	require.True(t, alias.Specs[0].Alias)
	require.Equal(t, "User", alias.Specs[0].Type.Name)

	main := file.Decls[6].(*Func)
	require.Equal(t, "main", main.Name)
	require.Equal(t, "fmt", main.Body[0].Expr.Fun.Path)
	require.Equal(t, "strconv", main.Body[1].RHS[0].Fun.Path)
	require.Equal(t, "Atoi", main.Body[1].RHS[0].Fun.Name)
	require.Equal(t, StmtIf, main.Body[2].Kind)
	require.Equal(t, ExprComposite, main.Body[3].RHS[0].Kind)
	require.Equal(t, OpSub, main.Body[5].RHS[0].Op)
	require.Equal(t, ExprIndex, main.Body[9].RHS[0].Kind)
	require.Equal(t, ExprSlice, main.Body[10].RHS[0].Kind)
	require.Equal(t, StmtFor, main.Body[14].Kind)
	require.Equal(t, StmtRange, main.Body[15].Kind)
	require.Equal(t, StmtRange, main.Body[16].Kind)
	require.Equal(t, BranchBreak, main.Body[16].Body[0].Branch)
	require.Equal(t, StmtDefer, main.Body[17].Kind)
	require.Equal(t, StmtGo, main.Body[18].Kind)
	require.Equal(t, StmtDecl, main.Body[19].Kind)
	require.Equal(t, StmtBlock, main.Body[23].Kind)
	require.Equal(t, ExprFunc, main.Body[24].RHS[0].Fun.Kind)

	method := file.Decls[7].(*Func)
	require.Equal(t, "u", method.Receiver.Names[0])
	require.Equal(t, TypePointer, method.Receiver.Type.Kind)
	add := file.Decls[8].(*Func)
	require.Equal(t, []string{"a", "b"}, add.Parameters[0].Names)
	echo := file.Decls[9].(*Func)
	require.True(t, echo.Parameters[0].Variadic)
	require.True(t, echo.Body[0].Expr.Ellipsis)
}

func TestShadowedPackage(t *testing.T) {
	t.Parallel()
	const src = `package p

import "fmt"

func f(fmt int) *fmt.Stringer {
	return nil
}

type T struct{}

func g(fmt *T) {
	fmt.M()
}
`
	file := roundTrip(t, src)
	f := file.Decls[0].(*Func)
	require.Equal(t, "fmt", f.Parameters[0].Names[0])
	require.Equal(t, "fmt", f.Returns[0].Type.Elem.Path)
	require.Equal(t, "Stringer", f.Returns[0].Type.Elem.Name)
	g := file.Decls[2].(*Func)
	call := g.Body[0].Expr
	require.Equal(t, ExprCall, call.Kind)
	require.Empty(t, call.Fun.Path)
	require.Equal(t, "fmt", call.Fun.X.Name)
	require.Equal(t, "M", call.Fun.Name)
}

func TestPackageOnly(t *testing.T) {
	t.Parallel()
	file := roundTrip(t, "package p\n")
	require.Equal(t, "p", file.Name)
	require.Empty(t, file.Imports)
	require.Empty(t, file.Decls)
}

func TestNilAndErrors(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, errs.ErrInvalid)
	require.ErrorIs(t, (*File)(nil).Make(io.Discard), errs.ErrInvalid)
	require.ErrorIs(t, NewFile("p").Make(nil), errs.ErrInvalid)

	_, err = Take(strings.NewReader("package p\nfunc {\n"))
	require.ErrorIs(t, err, errs.ErrSyntax)

	_, err = Take(strings.NewReader("package p\nfunc f() {\n\tgoto\n}\n"))
	require.ErrorIs(t, err, errs.ErrSyntax)
}

func TestSyntaxRoundTrip(t *testing.T) {
	t.Parallel()
	const src = `package p

import "fmt"

func f(x any, ch chan int) {
	switch x {
	case 1:
		fallthrough
	case 2, 3:
	default:
	}
	switch n := 0; n {
	case 1:
		_ = n
	}
	switch {
	case true:
		break
	}
	switch x.(type) {
	case int, *string:
		return
	case []byte:
	default:
	}
	switch fmt := x.(type) {
	case fmt.Stringer:
		_ = fmt
	}
	switch n := 0; x.(type) {
	case int:
		_ = n
	}
	select {
	case v := <-ch:
		_ = v
	case ch <- 1:
	default:
	}
	select {}
}

type List[T any] struct {
	Next *List[T]
}

type Pair[A, B any] struct {
	A A
	B B
}

func Map[T any, U ~[]T](x U) T {
	var zero T
	return zero
}

func (box *Box[T]) Get() T {
	return box.V
}

type Box[T any] struct{ V T }

type Num interface {
	~int | ~string
	fmt.Stringer
}

type Either[T any] interface {
	~int | ~[]T
}

func Use(x Pair[int, string], y List[int]) {
	_ = x
	_ = y
	var ready chan Pair[int, string]
	_ = ready
}
`
	file := roundTrip(t, src)
	fn := file.Decls[0].(*Func)
	require.Equal(t, StmtSwitch, fn.Body[0].Kind)
	require.Nil(t, fn.Body[0].Body[2].Exprs)
	require.Equal(t, StmtTypeSwitch, fn.Body[3].Kind)
	require.Equal(t, "fmt", fn.Body[4].Body[0].Exprs[0].Path)
	require.Equal(t, StmtSelect, fn.Body[6].Kind)
	require.Nil(t, fn.Body[6].Body[2].Stmt)
	list := file.Decls[1].(*TypeDecl)
	require.Equal(t, "T", list.Specs[0].Params[0].Names[0])
	require.Equal(t, TypePointer, list.Specs[0].Type.Fields[0].Type.Kind)
	require.Equal(t, TypeInst, list.Specs[0].Type.Fields[0].Type.Elem.Kind)
	num := file.Decls[6].(*TypeDecl)
	embedded := num.Specs[0].Type.Fields[0].Type
	require.Equal(t, TypeBinary, embedded.Kind)
	require.Equal(t, OpOr, embedded.Op)
	require.Equal(t, TypeUnary, embedded.Left.Kind)
	require.Equal(t, OpTilde, embedded.Left.Op)
}

func roundTrip(t *testing.T, src string) *File {
	t.Helper()
	first, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	second, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, first, second)
	return first
}

func roundTripFile(t *testing.T, file *File) *File {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return roundTrip(t, buf.String())
}
