package test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

// TestConventions rejects inline errors at return sites, testing assertions
// outside testify, nil contexts, comparisons of a context to nil, context
// roots outside main, a godoc example, the Cocoa click probe, and the
// Android host entry,
// sentinels that repeat a meta condition without wrapping it, an enum
// go:generate directive outside enum.go, and a raw OS name passed to
// ForGOOS or IncompatibleOff. The vendored compiler trees and the
// purego pin are not this module's code.
func TestConventions(t *testing.T) {
	root := moduleRoot(t)
	index := newSigIndex(modulePath(root))
	var problems []string
	var files []parsedFile
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := entry.Name()
		if entry.IsDir() {
			if skipConventionDir(name) || skippedCompilerTree(root, path) || skippedPuregoPin(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || skippedCompilerTree(root, path) || skippedPuregoPin(root, path) {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution|parser.ParseComments)
		if err != nil {
			problems = append(problems, rel+": "+err.Error())
			return nil
		}
		index.addFile(rel, file)
		files = append(files, parsedFile{name: rel, fset: fset, file: file})
		return nil
	})
	require.NoError(t, err)
	for _, file := range files {
		problems = append(problems, inspectFile(file, index)...)
	}
	require.Empty(t, problems)
}

func TestConventionChecker(t *testing.T) {
	const src = `package p

import (
	"errors"
	"fmt"
	"testing"
)

var ErrKnown = errors.New("known")

func pass(kind, err error) error {
	return fmt.Errorf("%w: %w", kind, err)
}

func wrapped(err error) error {
	return fmt.Errorf("open: %w", ErrKnown)
}

func inlineNew() error {
	return errors.New("nope")
}

func inlineErrorf(name string) error {
	return fmt.Errorf("open %s", name)
}

func TestBad(t *testing.T) {
	t.Fatal("x")
}
`
	got := checkSource("p.go", []byte(src))
	require.Equal(t, []string{
		"p.go:20:9: errors.New at a return",
		"p.go:24:9: fmt.Errorf at a return",
		"p.go:28:2: testing assertion",
	}, got)

	const testSrc = `package p

import (
	"errors"
	"testing"
)

func TestFixture(t *testing.T) error {
	return errors.New("boom")
}
`
	require.Empty(t, checkSource("p_test.go", []byte(testSrc)))

	const enums = `package p

const (
	StatusOK = "ok"
	StatusNo = "no"
	StatusYes = "yes"
)

const (
	red = "red"
	green = "green"
	blue = "blue"
)

type Kind string

const (
	KindA Kind = "a"
	KindB Kind = "b"
	KindC Kind = "c"
)

const (
	mimeHTML = "text/html"
	mimeText = "text/plain"
	mimeJSON = "application/json"
)
`
	require.Equal(t, []string{
		"p.go:4:2: untyped string const group looks like an enum",
		"p.go:10:2: untyped string const group looks like an enum",
	}, checkSource("p.go", []byte(enums)))
	require.Empty(t, checkSource("p_test.go", []byte(enums)))

	const open = `package p

const (
	alpineRelease = "3.24"
	alpineVersion = "6.18.56-r0"
	alpinePkg = "linux-virt"
)

const (
	envData = "LEWKIT_DATA_DIR"
	envCache = "LEWKIT_CACHE_DIR"
	envConfig = "LEWKIT_CONFIG_DIR"
)
`
	require.Empty(t, checkSource("p.go", []byte(open)))

	directive := "package p\n\n" +
		"//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum\n" +
		"//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate header foo.h\n"
	require.Equal(t, []string{
		"format.go:3:1: enum go:generate belongs in enum.go",
	}, checkSource("format.go", []byte(directive)))
	require.Empty(t, checkSource("enum.go", []byte(directive)))
	busy := "package p\n\n//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum\n\ntype Kind int\n"
	require.Equal(t, []string{
		"enum.go:5:1: enum.go holds only the package clause and the enum go:generate line",
	}, checkSource("enum.go", []byte(busy)))

	named := "package p\n\n//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate enum format.go\n"
	require.Equal(t, []string{
		"level.go:3:1: enum go:generate belongs in enum.go",
	}, checkSource("level.go", []byte(named)))

	const hostNames = `package p

func f() error {
	return driver.ForGOOS("linux")
}

func TestOff(t *testing.T) {
	test.IncompatibleOff(t, f, "ios")
}
`
	require.Equal(t, []string{
		"host.go:4:24: pass a GOOS enum, not a raw name",
		"host.go:8:29: pass a GOOS enum, not a raw name",
	}, checkSource("host.go", []byte(hostNames)))
	require.Empty(t, checkSource("host.go", []byte("package p\n\nfunc f() error {\n\treturn driver.ForGOOS(build.GOOSLinux)\n}\n")))
}

func TestConventionSkips(t *testing.T) {
	const src = `package p

import "testing"

func TestBad(t *testing.T) {
	t.Skip("x")
	t.Skipf("x %s", "y")
	t.SkipNow()
	b.Skip(err)
}
`
	require.Equal(t, []string{
		"p.go:6:2: testing assertion",
		"p.go:7:2: testing assertion",
		"p.go:8:2: testing assertion",
		"p.go:9:2: testing assertion",
	}, checkSource("p.go", []byte(src)))

	const helper = `package test

import "testing"

func Skip(tb testing.TB, args ...any) {
	tb.Helper()
	tb.Skip(args...)
}

func Skipf(tb testing.TB, format string, args ...any) {
	tb.Helper()
	tb.Skipf(format, args...)
}

func Need(tb testing.TB) {
	tb.Skipf("missing")
}
`
	require.Equal(t, []string{
		"x/test/test.go:16:2: testing assertion",
	}, checkSource("x/test/test.go", []byte(helper)))
}

func TestConventionContext(t *testing.T) {
	const src = `package p

import "context"

func takes(ctx context.Context) {}

func takesSlice(b []byte) {}

func notMain() {
	context.Background()
	context.TODO()
	takes(nil)
	takesSlice(nil)
	context.WithCancel(nil)
}

func main() {
	context.Background()
	takes(context.Background())
}
`
	require.Equal(t, []string{
		"p.go:10:2: context root",
		"p.go:11:2: context root",
		"p.go:12:8: nil context",
		"p.go:14:21: nil context",
	}, checkSource("p.go", []byte(src)))

	const example = `package p

import "context"

func Example() {
	context.Background()
}
`
	require.Empty(t, checkSource("p_test.go", []byte(example)))

	const probe = `package cocoa

import "context"

func probeContentClick() {
	context.Background()
}
`
	require.Empty(t, checkSource("x/driver/window/cocoa/mouse_darwin_test.go", []byte(probe)))
	require.Equal(t, []string{
		"p_test.go:6:2: context root",
	}, checkSource("p_test.go", []byte(probe)))
}

func TestConventionNilGuard(t *testing.T) {
	const src = `package p

import "context"

func takes(ctx context.Context) {
	if ctx == nil {
		return
	}
}

func parentNil(parent context.Context) {
	if parent != nil {
		_ = parent
	}
}

type hold struct {
	ctx context.Context
}

func (h hold) use() {
	if h.ctx == nil {
		return
	}
}

func buffer(ctx *int) {
	if ctx == nil {
	}
}
`
	require.Equal(t, []string{
		"p.go:6:12: nil context guard",
		"p.go:12:15: nil context guard",
		"p.go:22:14: nil context guard",
	}, checkSource("p.go", []byte(src)))
}

func TestConventionMeta(t *testing.T) {
	const src = `package p

import "errors"

var (
	ErrFile = errors.New("file not found")
	ErrNil  = errors.New("make: nil context")
	ErrKnown = errors.New("known")
)
`
	require.Equal(t, []string{
		"p.go:6:2: wrap errs.ErrNotFound",
		"p.go:7:2: use errs.ErrNilContext",
	}, checkSource("p.go", []byte(src)))

	const testSrc = `package p

import "errors"

var ErrFile = errors.New("file not found")
`
	require.Empty(t, checkSource("p_test.go", []byte(testSrc)))

	const meta = `package errs

import "errors"

var ErrNotFound = errors.New("not found")
`
	require.Empty(t, checkSource("x/errs/errs.go", []byte(meta)))
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent)
		dir = parent
	}
}

func skipConventionDir(name string) bool {
	switch name {
	case ".git", "vendor", "testdata", "_rewrite":
		return true
	default:
		return false
	}
}

func skippedCompilerTree(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "x/toolchain/go/_upstream" || strings.HasPrefix(rel, "x/toolchain/go/_upstream/") ||
		rel == "x/toolchain/go/src" || strings.HasPrefix(rel, "x/toolchain/go/src/")
}

// skippedPuregoPin is the modot place of github.com/ebitengine/purego.
// Upstream style is not this module's conventions.
func skippedPuregoPin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	return rel == "third_party/purego" || strings.HasPrefix(rel, "third_party/purego/")
}

func checkSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution|parser.ParseComments)
	if err != nil {
		return []string{name + ": " + err.Error()}
	}
	index := newSigIndex("")
	index.addFile(name, file)
	return inspectFile(parsedFile{name: name, fset: fset, file: file}, index)
}

type parsedFile struct {
	name string
	fset *token.FileSet
	file *ast.File
}

func inspectFile(file parsedFile, index *sigIndex) []string {
	if generatedFile(file.file) {
		return nil
	}
	checker := &conventionChecker{
		name:       file.name,
		fset:       file.fset,
		testFile:   strings.HasSuffix(file.name, "_test.go"),
		imports:    map[string]bool{},
		importPath: map[string]string{},
		errors:     "errors",
		fmt:        "fmt",
		index:      index,
		dir:        pkgDir(file.name),
	}
	checker.readImports(file.file)
	ast.Walk(checker, file.file)
	checker.problems = append(checker.problems, enumGenerateDirective(file)...)
	if !checker.testFile {
		checker.problems = append(checker.problems, stringConstEnums(file)...)
	}
	return checker.problems
}

// enumGenerateDirective flags an enum go:generate line outside enum.go.
// enum.go itself is the package clause and that line.
func enumGenerateDirective(file parsedFile) []string {
	var hits []*ast.Comment
	for _, group := range file.file.Comments {
		for _, comment := range group.List {
			if enumGenerateLine(comment.Text) {
				hits = append(hits, comment)
			}
		}
	}
	if len(hits) == 0 {
		return nil
	}
	if filepath.Base(file.name) != "enum.go" {
		problems := make([]string, 0, len(hits))
		for _, comment := range hits {
			pos := file.fset.Position(comment.Pos())
			problems = append(problems, fmt.Sprintf("%s:%d:%d: enum go:generate belongs in enum.go", file.name, pos.Line, pos.Column))
		}
		return problems
	}
	if len(file.file.Decls) == 0 {
		return nil
	}
	pos := file.fset.Position(file.file.Decls[0].Pos())
	return []string{fmt.Sprintf("%s:%d:%d: enum.go holds only the package clause and the enum go:generate line", file.name, pos.Line, pos.Column)}
}

func enumGenerateLine(text string) bool {
	text = strings.TrimSpace(strings.TrimPrefix(text, "//"))
	if !strings.HasPrefix(text, "go:generate") {
		return false
	}
	fields := strings.Fields(text)
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "generate" && fields[i+1] == "enum" {
			return true
		}
	}
	return false
}

type conventionChecker struct {
	name        string
	fset        *token.FileSet
	testFile    bool
	imports     map[string]bool
	importPath  map[string]string
	contextName string
	errors      string
	fmt         string
	index       *sigIndex
	dir         string
	stack       []ast.Node
	scopes      []nameScope
	problems    []string
}

type nameScope struct {
	depth int
	names map[string]string
}

const kindContext = "\x00context"

func (c *conventionChecker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		c.stack = c.stack[:len(c.stack)-1]
		if len(c.scopes) > 0 && c.scopes[len(c.scopes)-1].depth == len(c.stack) {
			c.scopes = c.scopes[:len(c.scopes)-1]
		}
		return nil
	}
	switch n := node.(type) {
	case *ast.FuncDecl:
		c.scopes = append(c.scopes, nameScope{depth: len(c.stack), names: c.bindFunc(n.Recv, n.Type)})
	case *ast.FuncLit:
		c.scopes = append(c.scopes, nameScope{depth: len(c.stack), names: c.bindFunc(nil, n.Type)})
	}
	c.inspect(node)
	c.stack = append(c.stack, node)
	return c
}

func (c *conventionChecker) inspect(node ast.Node) {
	c.noteSentinel(node)
	if bin, ok := node.(*ast.BinaryExpr); ok {
		c.noteNilGuard(bin)
	}
	call, ok := node.(*ast.CallExpr)
	if !ok {
		return
	}
	c.noteNilContext(call)
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	c.noteAssertion(call, selector)
	c.noteContextRoot(call, selector)
	c.noteHostEnum(call, selector)
	if c.testFile || !inReturn(c.stack) {
		return
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}
	switch {
	case receiver.Name == c.errors && selector.Sel.Name == "New":
		c.add(call, "errors.New at a return")
	case receiver.Name == c.fmt && selector.Sel.Name == "Errorf":
		if !allowedErrorf(call, c.stack) {
			c.add(call, "fmt.Errorf at a return")
		}
	}
}

// noteSentinel flags a package-level sentinel whose text is a meta condition
// but which does not wrap that parent. x/errs defines the parents.
func (c *conventionChecker) noteSentinel(node ast.Node) {
	spec, ok := node.(*ast.ValueSpec)
	if !ok || c.testFile || inFunc(c.stack) {
		return
	}
	for i, name := range spec.Names {
		if name == nil || i >= len(spec.Values) {
			continue
		}
		text, ok := sentinelText(spec.Values[i], c.errors, c.fmt)
		if !ok {
			continue
		}
		parent, collapse := errs.Classify(name.Name, text)
		if parent == "" {
			continue
		}
		if collapse && name.Name == parent && filepath.ToSlash(c.name) == "x/errs/errs.go" {
			continue
		}
		if collapse {
			c.add(name, "use errs."+parent)
			continue
		}
		c.add(name, "wrap errs."+parent)
	}
}

func sentinelText(expr ast.Expr, errorsName, fmtName string) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || len(call.Args) == 0 {
		return "", false
	}
	receiver, ok := selector.X.(*ast.Ident)
	if !ok {
		return "", false
	}
	switch {
	case receiver.Name == errorsName && selector.Sel.Name == "New" && len(call.Args) == 1:
		return stringLiteral(call.Args[0])
	case receiver.Name == fmtName && selector.Sel.Name == "Errorf" && len(call.Args) == 1:
		text, ok := stringLiteral(call.Args[0])
		if !ok || strings.Contains(text, "%") {
			return "", false
		}
		return text, true
	default:
		return "", false
	}
}

func inFunc(stack []ast.Node) bool {
	for _, node := range stack {
		switch node.(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return true
		}
	}
	return false
}

func (c *conventionChecker) noteAssertion(call *ast.CallExpr, selector *ast.SelectorExpr) {
	name := selector.Sel.Name
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || c.imports[receiver.Name] {
		return
	}
	switch name {
	case "Fatal", "Fatalf", "Fail", "FailNow":
		c.add(call, "testing assertion")
	case "Error", "Errorf":
		if name == "Error" && len(call.Args) == 0 {
			return
		}
		if testingReceiver(receiver.Name) {
			c.add(call, "testing assertion")
		}
	case "Skip", "Skipf", "SkipNow":
		if !testingReceiver(receiver.Name) || c.skipHelper() {
			return
		}
		c.add(call, "testing assertion")
	}
}

// noteHostEnum flags a raw OS name passed to ForGOOS or IncompatibleOff.
// Those arguments are build.GOOS values.
func (c *conventionChecker) noteHostEnum(call *ast.CallExpr, selector *ast.SelectorExpr) {
	switch selector.Sel.Name {
	case "ForGOOS", "IncompatibleOff":
	default:
		return
	}
	for _, arg := range call.Args {
		lit, ok := arg.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		c.add(lit, "pass a GOOS enum, not a raw name")
	}
}

// skipHelper is the body of test.Skip and test.Skipf, the one place a
// testing skip call remains.
func (c *conventionChecker) skipHelper() bool {
	if filepath.ToSlash(c.name) != "x/test/test.go" {
		return false
	}
	decl := enclosingFuncDecl(c.stack)
	if decl == nil || decl.Name == nil {
		return false
	}
	switch decl.Name.Name {
	case "Skip", "Skipf":
		return true
	default:
		return false
	}
}

func testingReceiver(name string) bool {
	switch name {
	case "t", "tb", "b", "tt":
		return true
	default:
		return false
	}
}

func (c *conventionChecker) noteContextRoot(call *ast.CallExpr, selector *ast.SelectorExpr) {
	receiver, ok := selector.X.(*ast.Ident)
	if !ok || c.contextName == "" || receiver.Name != c.contextName {
		return
	}
	if selector.Sel.Name != "Background" && selector.Sel.Name != "TODO" {
		return
	}
	if allowedContextRoot(enclosingFuncDecl(c.stack), c.name) {
		return
	}
	c.add(call, "context root")
}

func allowedContextRoot(decl *ast.FuncDecl, filename string) bool {
	if decl == nil || decl.Name == nil {
		return false
	}
	name := decl.Name.Name
	file := filepath.ToSlash(filename)
	switch {
	case name == "main":
		return true
	case strings.HasPrefix(name, "Example") && strings.HasSuffix(file, "_test.go"):
		return true
	case name == "runHost" && strings.HasSuffix(file, "x/entry/android/host.go"):
		return true
	case name == "probeContentClick" && strings.HasSuffix(file, "x/driver/window/cocoa/mouse_darwin_test.go"):
		return true
	default:
		return false
	}
}

func (c *conventionChecker) noteNilGuard(bin *ast.BinaryExpr) {
	if bin.Op != token.EQL && bin.Op != token.NEQ {
		return
	}
	var other, nilExpr ast.Expr
	switch {
	case isNilExpr(bin.X):
		other, nilExpr = bin.Y, bin.X
	case isNilExpr(bin.Y):
		other, nilExpr = bin.X, bin.Y
	default:
		return
	}
	if !c.exprIsContext(other) {
		return
	}
	c.add(nilExpr, "nil context guard")
}

func (c *conventionChecker) exprIsContext(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.Ident:
		kind, ok := c.lookup(e.Name)
		return ok && kind == kindContext
	case *ast.SelectorExpr:
		id, ok := e.X.(*ast.Ident)
		if !ok {
			return c.index.onlyContextField(c.dir, e.Sel.Name)
		}
		kind, known := c.lookup(id.Name)
		if known && kind != "" && kind != kindContext {
			return c.index.fieldIsContext(c.dir, kind, e.Sel.Name)
		}
		if known && kind == kindContext {
			return false
		}
		return c.index.onlyContextField(c.dir, e.Sel.Name)
	case *ast.ParenExpr:
		return c.exprIsContext(e.X)
	default:
		return false
	}
}

func (c *conventionChecker) lookup(name string) (string, bool) {
	for i := len(c.scopes) - 1; i >= 0; i-- {
		kind, ok := c.scopes[i].names[name]
		if ok {
			return kind, true
		}
	}
	return "", false
}

func (c *conventionChecker) bindFunc(recv *ast.FieldList, typ *ast.FuncType) map[string]string {
	names := map[string]string{}
	c.bindFields(names, recv)
	if typ != nil {
		c.bindFields(names, typ.Params)
	}
	return names
}

func (c *conventionChecker) bindFields(names map[string]string, list *ast.FieldList) {
	if list == nil {
		return
	}
	for _, field := range list.List {
		kind := c.valueKind(field.Type)
		for _, name := range field.Names {
			names[name.Name] = kind
		}
	}
}

func (c *conventionChecker) valueKind(expr ast.Expr) string {
	if c.contextName != "" && isContextType(expr, c.contextName) {
		return kindContext
	}
	name := typeIdent(expr)
	if name != "" && c.index.isStruct(c.dir, name) {
		return name
	}
	return ""
}

func (c *conventionChecker) noteNilContext(call *ast.CallExpr) {
	for i, arg := range call.Args {
		if !isNilExpr(arg) || !c.argIsContext(call, i) {
			continue
		}
		c.add(arg, "nil context")
	}
}

func isNilExpr(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name == "nil"
	case *ast.ParenExpr:
		return isNilExpr(value.X)
	default:
		return false
	}
}

func (c *conventionChecker) argIsContext(call *ast.CallExpr, index int) bool {
	switch fun := unwrapFun(call.Fun).(type) {
	case *ast.Ident:
		return c.index.contextAt(c.dir, fun.Name, index)
	case *ast.SelectorExpr:
		pkg, ok := fun.X.(*ast.Ident)
		if ok {
			if path, isImport := c.importPath[pkg.Name]; isImport {
				if path == "context" {
					return stdlibContextArg(fun.Sel.Name, index)
				}
				dir, known := c.index.dirForImport(path)
				return known && c.index.contextAt(dir, fun.Sel.Name, index)
			}
		}
		if dir, typeName, ok := concreteType(fun.X, c.importPath, c.dir, c.index); ok {
			return c.index.contextAt(dir, typeName+"."+fun.Sel.Name, index)
		}
		if c.index.contextAt(c.dir, "."+fun.Sel.Name, index) {
			return true
		}
		return c.index.uniqueMethod(fun.Sel.Name, index)
	default:
		return false
	}
}

func unwrapFun(expr ast.Expr) ast.Expr {
	switch value := expr.(type) {
	case *ast.IndexExpr:
		return unwrapFun(value.X)
	case *ast.IndexListExpr:
		return unwrapFun(value.X)
	case *ast.ParenExpr:
		return unwrapFun(value.X)
	default:
		return expr
	}
}

func concreteType(expr ast.Expr, imports map[string]string, fileDir string, index *sigIndex) (string, string, bool) {
	switch value := expr.(type) {
	case *ast.ParenExpr:
		return concreteType(value.X, imports, fileDir, index)
	case *ast.StarExpr:
		return concreteType(value.X, imports, fileDir, index)
	case *ast.UnaryExpr:
		return concreteType(value.X, imports, fileDir, index)
	case *ast.CompositeLit:
		return namedType(value.Type, imports, fileDir, index)
	default:
		return "", "", false
	}
}

func namedType(expr ast.Expr, imports map[string]string, fileDir string, index *sigIndex) (string, string, bool) {
	switch value := expr.(type) {
	case *ast.Ident:
		return fileDir, value.Name, true
	case *ast.StarExpr:
		return namedType(value.X, imports, fileDir, index)
	case *ast.IndexExpr:
		return namedType(value.X, imports, fileDir, index)
	case *ast.IndexListExpr:
		return namedType(value.X, imports, fileDir, index)
	case *ast.SelectorExpr:
		pkg, ok := value.X.(*ast.Ident)
		if !ok {
			return "", "", false
		}
		path, isImport := imports[pkg.Name]
		if !isImport {
			return "", "", false
		}
		dir, known := index.dirForImport(path)
		if !known {
			return "", "", false
		}
		return dir, value.Sel.Name, true
	default:
		return "", "", false
	}
}

func stdlibContextArg(name string, index int) bool {
	if index != 0 {
		return false
	}
	switch name {
	case "WithCancel", "WithDeadline", "WithTimeout", "WithValue",
		"WithCancelCause", "WithDeadlineCause", "WithTimeoutCause",
		"AfterFunc", "Cause", "WithoutCancel":
		return true
	default:
		return false
	}
}

type sigIndex struct {
	mod          string
	funcs        map[string][][]bool
	methods      map[string][]methodSig
	structs      map[string]bool
	fieldContext map[string]bool
	fieldMixed   map[string]bool
}

type methodSig struct {
	recv string
	mask []bool
}

func newSigIndex(mod string) *sigIndex {
	return &sigIndex{
		mod:          mod,
		funcs:        map[string][][]bool{},
		methods:      map[string][]methodSig{},
		structs:      map[string]bool{},
		fieldContext: map[string]bool{},
		fieldMixed:   map[string]bool{},
	}
}

func (idx *sigIndex) addFile(rel string, file *ast.File) {
	dir := pkgDir(rel)
	contextName := contextImportName(file)
	idx.addFields(dir, file, contextName)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Type == nil {
			continue
		}
		mask := contextMask(fn.Type, contextName)
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recv := typeIdent(fn.Recv.List[0].Type)
			if recv == "" {
				continue
			}
			idx.funcs[dir+"\x00"+recv+"."+fn.Name.Name] = append(idx.funcs[dir+"\x00"+recv+"."+fn.Name.Name], mask)
			idx.funcs[dir+"\x00"+"."+fn.Name.Name] = append(idx.funcs[dir+"\x00"+"."+fn.Name.Name], mask)
			idx.methods[fn.Name.Name] = append(idx.methods[fn.Name.Name], methodSig{recv: dir + "." + recv, mask: mask})
			continue
		}
		idx.funcs[dir+"\x00"+fn.Name.Name] = append(idx.funcs[dir+"\x00"+fn.Name.Name], mask)
	}
}

func (idx *sigIndex) addFields(dir string, file *ast.File, contextName string) {
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			idx.structs[dir+"\x00"+ts.Name.Name] = true
			if st.Fields == nil {
				continue
			}
			for _, field := range st.Fields.List {
				isContext := contextName != "" && isContextType(field.Type, contextName)
				for _, name := range field.Names {
					key := dir + "\x00" + ts.Name.Name + "." + name.Name
					prev, seen := idx.fieldContext[key]
					if !seen {
						idx.fieldContext[key] = isContext
						continue
					}
					if prev != isContext {
						idx.fieldContext[key] = false
						idx.fieldMixed[key] = true
					}
				}
			}
		}
	}
}

func (idx *sigIndex) isStruct(dir, name string) bool {
	return idx.structs[dir+"\x00"+name]
}

func (idx *sigIndex) fieldIsContext(dir, typ, name string) bool {
	key := dir + "\x00" + typ + "." + name
	return idx.fieldContext[key] && !idx.fieldMixed[key]
}

func (idx *sigIndex) onlyContextField(dir, name string) bool {
	prefix := dir + "\x00"
	suffix := "." + name
	found := false
	for key, isContext := range idx.fieldContext {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if !strings.HasSuffix(key[len(prefix):], suffix) {
			continue
		}
		if idx.fieldMixed[key] || !isContext {
			return false
		}
		found = true
	}
	return found
}

func (idx *sigIndex) contextAt(dir, name string, index int) bool {
	return masksAgree(idx.funcs[dir+"\x00"+name], index)
}

func (idx *sigIndex) uniqueMethod(name string, index int) bool {
	sigs := idx.methods[name]
	if len(sigs) == 0 {
		return false
	}
	recv := sigs[0].recv
	for _, sig := range sigs {
		if sig.recv != recv || index >= len(sig.mask) || !sig.mask[index] {
			return false
		}
	}
	return true
}

func (idx *sigIndex) dirForImport(path string) (string, bool) {
	if idx.mod == "" {
		return "", false
	}
	if path == idx.mod {
		return "", true
	}
	prefix := idx.mod + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	return path[len(prefix):], true
}

func masksAgree(masks [][]bool, index int) bool {
	if len(masks) == 0 {
		return false
	}
	for _, mask := range masks {
		if index >= len(mask) || !mask[index] {
			return false
		}
	}
	return true
}

func contextImportName(file *ast.File) string {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil || path != "context" {
			continue
		}
		if spec.Name != nil && spec.Name.Name != "_" && spec.Name.Name != "." {
			return spec.Name.Name
		}
		return "context"
	}
	return ""
}

func contextMask(fn *ast.FuncType, contextName string) []bool {
	if fn.Params == nil {
		return nil
	}
	var mask []bool
	for _, field := range fn.Params.List {
		isContext := contextName != "" && isContextType(field.Type, contextName)
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			mask = append(mask, isContext)
		}
	}
	return mask
}

func isContextType(expr ast.Expr, contextName string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == contextName && selector.Sel.Name == "Context"
}

func typeIdent(expr ast.Expr) string {
	switch value := expr.(type) {
	case *ast.Ident:
		return value.Name
	case *ast.StarExpr:
		return typeIdent(value.X)
	case *ast.IndexExpr:
		return typeIdent(value.X)
	case *ast.IndexListExpr:
		return typeIdent(value.X)
	case *ast.SelectorExpr:
		return value.Sel.Name
	default:
		return ""
	}
}

func pkgDir(rel string) string {
	dir := filepath.ToSlash(filepath.Dir(rel))
	if dir == "." {
		return ""
	}
	return dir
}

func modulePath(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "module" {
			return fields[1]
		}
	}
	return ""
}

func (c *conventionChecker) add(node ast.Node, message string) {
	pos := c.fset.Position(node.Pos())
	c.problems = append(c.problems, pos.Filename+":"+strconv.Itoa(pos.Line)+":"+strconv.Itoa(pos.Column)+": "+message)
}

func (c *conventionChecker) readImports(file *ast.File) {
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}
		local := path
		if slash := strings.LastIndex(path, "/"); slash >= 0 {
			local = path[slash+1:]
		}
		if spec.Name != nil {
			local = spec.Name.Name
		}
		if local == "_" || local == "." {
			continue
		}
		c.imports[local] = true
		c.importPath[local] = path
		if path == "errors" {
			c.errors = local
		}
		if path == "fmt" {
			c.fmt = local
		}
		if path == "context" {
			c.contextName = local
		}
	}
}

// stringConstEnums flags a parenthesized const block of untyped string
// literals that names a closed set. A typed enum is not flagged.
func stringConstEnums(file parsedFile) []string {
	var problems []string
	for _, decl := range file.file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST || !gen.Lparen.IsValid() {
			continue
		}
		specs, ok := untypedStringSpecs(gen)
		if !ok || len(specs) < 3 {
			continue
		}
		names := make([]string, len(specs))
		singleWords := true
		for i, vs := range specs {
			names[i] = vs.Names[0].Name
			text, err := strconv.Unquote(vs.Values[0].(*ast.BasicLit).Value)
			if err != nil || !closedSetToken(text) {
				singleWords = false
				names = nil
				break
			}
			if !oneWord(text) {
				singleWords = false
			}
		}
		if names == nil {
			continue
		}
		if !singleWords && !sharedEnumPrefix(names) {
			continue
		}
		pos := file.fset.Position(specs[0].Pos())
		problems = append(problems, fmt.Sprintf("%s:%d:%d: untyped string const group looks like an enum", file.name, pos.Line, pos.Column))
	}
	return problems
}

func untypedStringSpecs(gen *ast.GenDecl) ([]*ast.ValueSpec, bool) {
	specs := make([]*ast.ValueSpec, 0, len(gen.Specs))
	for _, spec := range gen.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok || vs.Type != nil || len(vs.Names) != 1 || len(vs.Values) != 1 {
			return nil, false
		}
		lit, ok := vs.Values[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return nil, false
		}
		specs = append(specs, vs)
	}
	return specs, true
}

func closedSetToken(text string) bool {
	if text == "" || strings.Contains(text, " ") || strings.Contains(text, "/") || strings.Contains(text, ".") {
		return false
	}
	for _, r := range text {
		if unicode.IsLower(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '&' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func oneWord(text string) bool {
	if text == "" {
		return false
	}
	for _, r := range text {
		if !unicode.IsLower(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	r, _ := utf8.DecodeRuneInString(text)
	return unicode.IsLetter(r)
}

func sharedEnumPrefix(names []string) bool {
	prefix := names[0]
	for _, name := range names[1:] {
		for prefix != "" && !strings.HasPrefix(name, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	if len(prefix) < 2 {
		return false
	}
	longer := false
	for _, name := range names {
		if len(name) == len(prefix) {
			continue
		}
		longer = true
		r, _ := utf8.DecodeRuneInString(name[len(prefix):])
		if !unicode.IsUpper(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return longer
}

func generatedFile(file *ast.File) bool {
	for _, group := range file.Comments {
		text := group.Text()
		if strings.Contains(text, "Code generated") || strings.Contains(text, "DO NOT EDIT") {
			return true
		}
	}
	return false
}

func inReturn(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.ReturnStmt:
			return true
		case *ast.FuncLit, *ast.FuncDecl:
			return false
		}
	}
	return false
}

func enclosingFunc(stack []ast.Node) ast.Node {
	for i := len(stack) - 1; i >= 0; i-- {
		switch stack[i].(type) {
		case *ast.FuncLit, *ast.FuncDecl:
			return stack[i]
		}
	}
	return nil
}

func enclosingFuncDecl(stack []ast.Node) *ast.FuncDecl {
	for i := len(stack) - 1; i >= 0; i-- {
		decl, ok := stack[i].(*ast.FuncDecl)
		if ok {
			return decl
		}
	}
	return nil
}

func allowedErrorf(call *ast.CallExpr, stack []ast.Node) bool {
	if len(call.Args) == 0 {
		return false
	}
	format, ok := stringLiteral(call.Args[0])
	if !ok {
		return false
	}
	if wrapsSentinel(format, call) {
		return true
	}
	return passThrough(format, call, enclosingFunc(stack))
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func wrapsSentinel(format string, call *ast.CallExpr) bool {
	for _, arg := range verbArgs(format, call, 'w') {
		if sentinelExpr(arg) {
			return true
		}
	}
	return false
}

func sentinelExpr(expr ast.Expr) bool {
	switch value := expr.(type) {
	case *ast.Ident:
		return sentinelName(value.Name)
	case *ast.SelectorExpr:
		return sentinelName(value.Sel.Name)
	default:
		return false
	}
}

func sentinelName(name string) bool {
	if name == "err" || name == "Err" {
		return false
	}
	if strings.HasPrefix(name, "Err") && len(name) > 3 && sentinelSuffix(name[3:]) {
		return true
	}
	return strings.HasPrefix(name, "err") && len(name) > 3 && sentinelSuffix(name[3:])
}

func sentinelSuffix(rest string) bool {
	first, _ := utf8.DecodeRuneInString(rest)
	return unicode.IsUpper(first) || rest[0] == '_'
}

func passThrough(format string, call *ast.CallExpr, fn ast.Node) bool {
	if formatHasLetter(format) {
		return false
	}
	params := errorParams(fn)
	verbs := verbArgs(format, call, 'w')
	if len(verbs) == 0 {
		return false
	}
	sawKind := false
	for _, arg := range verbs {
		id, ok := arg.(*ast.Ident)
		if !ok {
			return false
		}
		if sentinelName(id.Name) {
			sawKind = true
			continue
		}
		if params[id.Name] && id.Name != "err" {
			sawKind = true
			continue
		}
		if params[id.Name] && sawKind {
			continue
		}
		return false
	}
	return sawKind
}

func errorParams(fn ast.Node) map[string]bool {
	out := map[string]bool{}
	var funcType *ast.FuncType
	switch typed := fn.(type) {
	case *ast.FuncDecl:
		funcType = typed.Type
	case *ast.FuncLit:
		funcType = typed.Type
	default:
		return out
	}
	if funcType == nil || funcType.Params == nil {
		return out
	}
	for _, field := range funcType.Params.List {
		ident, ok := field.Type.(*ast.Ident)
		if !ok || ident.Name != "error" {
			continue
		}
		for _, name := range field.Names {
			out[name.Name] = true
		}
	}
	return out
}

func formatHasLetter(format string) bool {
	segments, err := splitFormat(format)
	if err != nil {
		return true
	}
	for _, segment := range segments {
		if !segment.literal {
			continue
		}
		if letterCore(segment.text) {
			return true
		}
	}
	return false
}

func letterCore(raw string) bool {
	i := 0
	for i < len(raw) {
		r, size := utf8.DecodeRuneInString(raw[i:])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			break
		}
		i += size
	}
	j := len(raw)
	for j > i {
		r, size := utf8.DecodeLastRuneInString(raw[:j])
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			break
		}
		j -= size
	}
	for _, r := range raw[i:j] {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

type formatSegment struct {
	literal bool
	text    string
}

func splitFormat(format string) ([]formatSegment, error) {
	var segments []formatSegment
	var literal strings.Builder
	flush := func() {
		if literal.Len() == 0 {
			return
		}
		segments = append(segments, formatSegment{literal: true, text: literal.String()})
		literal.Reset()
	}
	for i := 0; i < len(format); i++ {
		if format[i] != '%' {
			literal.WriteByte(format[i])
			continue
		}
		if i+1 >= len(format) {
			return nil, errBadFormat
		}
		if format[i+1] == '%' {
			literal.WriteString("%%")
			i++
			continue
		}
		j := i + 1
		for j < len(format) && strings.ContainsRune(" #+-0", rune(format[j])) {
			j++
		}
		if j < len(format) && (format[j] == '[' || format[j] == '*') {
			return nil, errBadFormat
		}
		for j < len(format) && format[j] >= '0' && format[j] <= '9' {
			j++
		}
		if j < len(format) && format[j] == '.' {
			j++
			if j < len(format) && format[j] == '*' {
				return nil, errBadFormat
			}
			for j < len(format) && format[j] >= '0' && format[j] <= '9' {
				j++
			}
		}
		if j >= len(format) || !strings.ContainsRune("vT%tbcdoOqxXUbeEfFgGspw", rune(format[j])) {
			return nil, errBadFormat
		}
		flush()
		segments = append(segments, formatSegment{text: format[i : j+1]})
		i = j
	}
	flush()
	return segments, nil
}

var errBadFormat = strconv.ErrSyntax

func verbArgs(format string, call *ast.CallExpr, verb byte) []ast.Expr {
	segments, err := splitFormat(format)
	if err != nil {
		return nil
	}
	var matched []ast.Expr
	argIndex := 0
	for _, segment := range segments {
		if segment.literal {
			continue
		}
		if argIndex+1 >= len(call.Args) {
			break
		}
		if segment.text[len(segment.text)-1] == verb {
			matched = append(matched, call.Args[argIndex+1])
		}
		argIndex++
	}
	return matched
}
