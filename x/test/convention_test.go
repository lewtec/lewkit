package test

import (
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

	"github.com/stretchr/testify/require"
)

// TestConventions rejects inline errors at return sites, testing assertions
// outside testify, nil contexts, and context roots outside main, a godoc
// example, and the Android host entry. The vendored compiler trees are not
// this module's code.
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
			if skipConventionDir(name) || skippedCompilerTree(root, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") || skippedCompilerTree(root, path) {
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
	return checker.problems
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
	problems    []string
}

func (c *conventionChecker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		c.stack = c.stack[:len(c.stack)-1]
		return nil
	}
	c.inspect(node)
	c.stack = append(c.stack, node)
	return c
}

func (c *conventionChecker) inspect(node ast.Node) {
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
	default:
		return false
	}
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
	mod     string
	funcs   map[string][][]bool
	methods map[string][]methodSig
}

type methodSig struct {
	recv string
	mask []bool
}

func newSigIndex(mod string) *sigIndex {
	return &sigIndex{
		mod:     mod,
		funcs:   map[string][][]bool{},
		methods: map[string][]methodSig{},
	}
}

func (idx *sigIndex) addFile(rel string, file *ast.File) {
	dir := pkgDir(rel)
	contextName := contextImportName(file)
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
