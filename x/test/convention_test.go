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

// TestConventions rejects inline errors at return sites and testing.T assertions
// outside testify. The vendored compiler trees are not this module's code.
func TestConventions(t *testing.T) {
	root := moduleRoot(t)
	var problems []string
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
		problems = append(problems, checkSource(rel, src)...)
		return nil
	})
	require.NoError(t, err)
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
	case ".git", "vendor", "testdata":
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
	if generatedFile(file) {
		return nil
	}
	checker := &conventionChecker{
		name:     name,
		fset:     fset,
		testFile: strings.HasSuffix(name, "_test.go"),
		imports:  map[string]bool{},
		errors:   "errors",
		fmt:      "fmt",
	}
	checker.readImports(file)
	ast.Walk(checker, file)
	return checker.problems
}

type conventionChecker struct {
	name     string
	fset     *token.FileSet
	testFile bool
	imports  map[string]bool
	errors   string
	fmt      string
	stack    []ast.Node
	problems []string
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
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	c.noteAssertion(call, selector)
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
		if path == "errors" {
			c.errors = local
		}
		if path == "fmt" {
			c.fmt = local
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
