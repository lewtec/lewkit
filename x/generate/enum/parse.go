package enum

import (
	"fmt"
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"
)

type fileSpec struct {
	pkg   string
	build string
	enums []enumSpec
}

type enumSpec struct {
	name       string
	stringKind bool
	fold       bool
	empty      string
	invalid    string
	hasInvalid bool
	emitConst  bool
	members    []member
}

type member struct {
	name    string
	token   string
	aliases []string
	value   constant.Value
}

type directive struct {
	fold       bool
	empty      string
	invalid    string
	hasInvalid bool
	hasValues  bool
	values     []string
}

func parseSource(path string, src []byte) (fileSpec, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return fileSpec{}, fmt.Errorf("%w: %s: %w", ErrEnum, path, err)
	}
	if file.Name == nil || !token.IsIdentifier(file.Name.Name) || token.IsKeyword(file.Name.Name) {
		return fileSpec{}, fmt.Errorf("%w: %s: package name", ErrEnum, path)
	}
	marks, err := markedTypes(file)
	if err != nil {
		return fileSpec{}, err
	}
	members, err := constMembers(file, marks)
	if err != nil {
		return fileSpec{}, err
	}
	var enums []enumSpec
	for _, name := range markOrder(file, marks) {
		mark := marks[name]
		got := members[name]
		list, emit, err := membersFor(name, mark, got)
		if err != nil {
			return fileSpec{}, err
		}
		if err := checkMembers(name, mark, list); err != nil {
			return fileSpec{}, err
		}
		enums = append(enums, enumSpec{
			name:       name,
			stringKind: mark.stringKind,
			fold:       mark.directive.fold,
			empty:      mark.directive.empty,
			invalid:    mark.directive.invalid,
			hasInvalid: mark.directive.hasInvalid,
			emitConst:  emit,
			members:    list,
		})
	}
	return fileSpec{pkg: file.Name.Name, build: leadingBuild(string(src)), enums: enums}, nil
}

type mark struct {
	stringKind bool
	directive  directive
}

func markedTypes(file *ast.File) (map[string]mark, error) {
	out := map[string]mark{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			ts := spec.(*ast.TypeSpec)
			dir, ok, err := enumDirective(gen, ts)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if ts.Assign.IsValid() {
				return nil, fmt.Errorf("%w: %s is an alias", ErrEnum, ts.Name.Name)
			}
			kind, ok := underlying(ts.Type)
			if !ok {
				return nil, fmt.Errorf("%w: %s is not an integer or string", ErrEnum, ts.Name.Name)
			}
			if _, dup := out[ts.Name.Name]; dup {
				return nil, fmt.Errorf("%w: %s declared twice", ErrEnum, ts.Name.Name)
			}
			out[ts.Name.Name] = mark{stringKind: kind == "string", directive: dir}
		}
	}
	return out, nil
}

func markOrder(file *ast.File, marks map[string]mark) []string {
	var names []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.TYPE {
			continue
		}
		for _, spec := range gen.Specs {
			name := spec.(*ast.TypeSpec).Name.Name
			if _, ok := marks[name]; ok {
				names = append(names, name)
			}
		}
	}
	return names
}

func enumDirective(gen *ast.GenDecl, ts *ast.TypeSpec) (directive, bool, error) {
	var lines []string
	if len(gen.Specs) == 1 {
		lines = append(lines, commentLines(gen.Doc)...)
	}
	lines = append(lines, commentLines(ts.Doc, ts.Comment)...)
	var dir directive
	found := false
	for _, line := range lines {
		if line != "lewkit:enum" && !strings.HasPrefix(line, "lewkit:enum ") {
			continue
		}
		if found {
			return directive{}, false, fmt.Errorf("%w: %s has two marks", ErrEnum, ts.Name.Name)
		}
		parsed, err := parseDirective(line)
		if err != nil {
			return directive{}, false, err
		}
		dir = parsed
		found = true
	}
	if !found {
		return directive{}, false, nil
	}
	return dir, true, nil
}

// commentLines reads comment text without CommentGroup.Text.
// Text drops //name:value lines, and the enum mark is one of those.
func commentLines(groups ...*ast.CommentGroup) []string {
	var lines []string
	for _, group := range groups {
		if group == nil {
			continue
		}
		for _, comment := range group.List {
			text := comment.Text
			if strings.HasPrefix(text, "//") {
				text = text[2:]
			}
			text = strings.TrimSpace(text)
			if text != "" {
				lines = append(lines, text)
			}
		}
	}
	return lines
}

func parseDirective(line string) (directive, error) {
	var dir directive
	fields := strings.Fields(line)
	for i := 1; i < len(fields); i++ {
		field := fields[i]
		switch {
		case field == "fold":
			dir.fold = true
		case strings.HasPrefix(field, "empty="):
			dir.empty = strings.TrimPrefix(field, "empty=")
			if dir.empty == "" {
				return directive{}, fmt.Errorf("%w: empty option", ErrEnum)
			}
		case strings.HasPrefix(field, "invalid="):
			dir.invalid = strings.TrimPrefix(field, "invalid=")
			dir.hasInvalid = true
			if dir.invalid == "" {
				return directive{}, fmt.Errorf("%w: invalid option", ErrEnum)
			}
		case strings.HasPrefix(field, "values="):
			if dir.hasValues {
				return directive{}, fmt.Errorf("%w: values option", ErrEnum)
			}
			raw := strings.TrimPrefix(field, "values=")
			for i+1 < len(fields) && !optionField(fields[i+1]) {
				i++
				raw += fields[i]
			}
			values, err := splitValues(raw)
			if err != nil {
				return directive{}, err
			}
			dir.values = values
			dir.hasValues = true
		default:
			return directive{}, fmt.Errorf("%w: %s", ErrEnum, field)
		}
	}
	return dir, nil
}

func optionField(field string) bool {
	return field == "fold" || strings.HasPrefix(field, "empty=") || strings.HasPrefix(field, "invalid=") || strings.HasPrefix(field, "values=")
}

func splitValues(raw string) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: values option", ErrEnum)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !tokenPattern(part) {
			return nil, fmt.Errorf("%w: values %s", ErrEnum, part)
		}
		if _, ok := seen[part]; ok {
			return nil, fmt.Errorf("%w: values %s", ErrEnum, part)
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out, nil
}

func membersFor(typeName string, mark mark, got []member) ([]member, bool, error) {
	if !mark.directive.hasValues {
		if len(got) == 0 {
			return nil, false, fmt.Errorf("%w: %s has no members", ErrEnum, typeName)
		}
		return got, false, nil
	}
	declared, err := declaredMembers(typeName, mark.directive.values)
	if err != nil {
		return nil, false, err
	}
	if len(got) == 0 {
		return declared, true, nil
	}
	if err := sameMembers(typeName, declared, got, mark.stringKind); err != nil {
		return nil, false, err
	}
	return declared, false, nil
}

func declaredMembers(typeName string, values []string) ([]member, error) {
	names := map[string]struct{}{}
	out := make([]member, 0, len(values))
	for _, tokenText := range values {
		name, err := goName(typeName, tokenText)
		if err != nil {
			return nil, err
		}
		if _, ok := names[name]; ok {
			return nil, fmt.Errorf("%w: %s value %s", ErrEnum, typeName, tokenText)
		}
		names[name] = struct{}{}
		out = append(out, member{name: name, token: tokenText})
	}
	return out, nil
}

func goName(typeName, tokenText string) (string, error) {
	ident := goIdent(tokenText)
	name := typeName + ident
	if ident == "" || !token.IsIdentifier(ident) || token.IsKeyword(ident) || !token.IsIdentifier(name) || token.IsKeyword(name) {
		return "", fmt.Errorf("%w: %s value %s", ErrEnum, typeName, tokenText)
	}
	return name, nil
}

func goIdent(text string) string {
	var b strings.Builder
	upper := true
	for _, r := range text {
		switch {
		case r == '_':
			b.WriteByte('_')
			upper = true
		case r == '-':
			upper = true
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func sameMembers(typeName string, declared, got []member, stringKind bool) error {
	if len(got) != len(declared) {
		return fmt.Errorf("%w: %s has %d members, values lists %d", ErrEnum, typeName, len(got), len(declared))
	}
	for i := range declared {
		want, have := declared[i], got[i]
		if have.name != want.name || have.token != want.token || len(have.aliases) != 0 {
			return fmt.Errorf("%w: %s member %s", ErrEnum, typeName, want.token)
		}
		if stringKind {
			if have.value.Kind() != constant.String || constant.StringVal(have.value) != want.token {
				return fmt.Errorf("%w: %s member %s", ErrEnum, typeName, want.token)
			}
			continue
		}
		if !constant.Compare(have.value, token.EQL, constant.MakeInt64(int64(i))) {
			return fmt.Errorf("%w: %s member %s", ErrEnum, typeName, want.token)
		}
	}
	return nil
}

func underlying(expr ast.Expr) (string, bool) {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return "", false
	}
	switch id.Name {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"byte", "rune":
		return id.Name, true
	default:
		return "", false
	}
}

func constMembers(file *ast.File, marks map[string]mark) (map[string][]member, error) {
	env := map[string]constant.Value{}
	out := map[string][]member{}
	seenVal := map[string][]constant.Value{}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		var prevType string
		var prev []ast.Expr
		for index, spec := range gen.Specs {
			vs := spec.(*ast.ValueSpec)
			values := vs.Values
			typeName := ""
			if len(values) == 0 {
				values = prev
				typeName = prevType
			}
			if vs.Type != nil {
				id, ok := vs.Type.(*ast.Ident)
				if !ok {
					typeName = ""
				} else {
					typeName = id.Name
				}
			}
			if len(vs.Values) > 0 {
				prev = vs.Values
				prevType = typeName
			}
			if len(values) == 0 || len(values) != len(vs.Names) {
				return nil, fmt.Errorf("%w: const list", ErrEnum)
			}
			iotaVal := constant.MakeInt64(int64(index))
			for i, name := range vs.Names {
				val, err := eval(values[i], iotaVal, env)
				if err != nil {
					return nil, err
				}
				if name.Name != "_" {
					if _, exists := env[name.Name]; exists {
						return nil, fmt.Errorf("%w: %s declared twice", ErrEnum, name.Name)
					}
					env[name.Name] = val
				}
				mark, wanted := marks[typeName]
				if !wanted || name.Name == "_" {
					continue
				}
				for _, prev := range seenVal[typeName] {
					if constant.Compare(prev, token.EQL, val) {
						return nil, fmt.Errorf("%w: %s repeats a value", ErrEnum, name.Name)
					}
				}
				seenVal[typeName] = append(seenVal[typeName], val)
				item, err := memberOf(typeName, name.Name, val, mark, vs.Comment)
				if err != nil {
					return nil, err
				}
				out[typeName] = append(out[typeName], item)
			}
		}
	}
	return out, nil
}

func memberOf(typeName, name string, val constant.Value, mark mark, comment *ast.CommentGroup) (member, error) {
	var tokenText string
	if mark.stringKind {
		if val.Kind() != constant.String {
			return member{}, fmt.Errorf("%w: %s is not a string", ErrEnum, name)
		}
		tokenText = constant.StringVal(val)
	} else {
		if val.Kind() != constant.Int {
			return member{}, fmt.Errorf("%w: %s is not an integer", ErrEnum, name)
		}
		tokenText = derivedToken(typeName, name)
	}
	if tokenText == "" {
		return member{}, fmt.Errorf("%w: %s has an empty token", ErrEnum, name)
	}
	aliases := aliasComment(comment)
	if !mark.stringKind && len(aliases) > 0 {
		tokenText = aliases[0]
		aliases = aliases[1:]
	}
	var extra []string
	for _, alias := range aliases {
		if alias == tokenText {
			continue
		}
		extra = append(extra, alias)
	}
	return member{name: name, token: tokenText, aliases: extra, value: val}, nil
}

func aliasComment(group *ast.CommentGroup) []string {
	lines := commentLines(group)
	if len(lines) != 1 {
		return nil
	}
	text := lines[0]
	if strings.Contains(text, "\n") {
		return nil
	}
	parts := strings.Split(text, ",")
	tokens := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !tokenPattern(part) {
			return nil
		}
		tokens = append(tokens, part)
	}
	return tokens
}

func tokenPattern(text string) bool {
	if text == "" || strings.Contains(text, " ") {
		return false
	}
	r, size := utf8.DecodeRuneInString(text)
	if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		return false
	}
	for _, r := range text[size:] {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '.' || r == '&' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func derivedToken(typeName, name string) string {
	rest := name
	if strings.HasPrefix(name, typeName) && len(name) > len(typeName) {
		rest = name[len(typeName):]
	}
	return strings.ToLower(rest)
}

func checkMembers(typeName string, mark mark, members []member) error {
	seen := map[string]string{}
	var emptyOK bool
	for _, item := range members {
		names := append([]string{item.token}, item.aliases...)
		for _, text := range names {
			key := text
			if mark.directive.fold {
				key = strings.ToLower(text)
			}
			if prev, ok := seen[key]; ok {
				return fmt.Errorf("%w: %s token %s repeats %s", ErrEnum, typeName, text, prev)
			}
			seen[key] = item.name
		}
		if item.token == mark.directive.empty {
			emptyOK = true
		}
	}
	if mark.directive.empty != "" && !emptyOK {
		return fmt.Errorf("%w: %s empty=%s", ErrEnum, typeName, mark.directive.empty)
	}
	return nil
}

func eval(expr ast.Expr, iota constant.Value, env map[string]constant.Value) (constant.Value, error) {
	switch e := expr.(type) {
	case *ast.Ident:
		if e.Name == "iota" {
			return iota, nil
		}
		val, ok := env[e.Name]
		if !ok {
			return nil, fmt.Errorf("%w: unknown const %s", ErrEnum, e.Name)
		}
		return val, nil
	case *ast.BasicLit:
		val := constant.MakeFromLiteral(e.Value, e.Kind, 0)
		if val.Kind() == constant.Unknown {
			return nil, fmt.Errorf("%w: literal %s", ErrEnum, e.Value)
		}
		return val, nil
	case *ast.ParenExpr:
		return eval(e.X, iota, env)
	case *ast.UnaryExpr:
		val, err := eval(e.X, iota, env)
		if err != nil {
			return nil, err
		}
		out := constant.UnaryOp(e.Op, val, 0)
		if out.Kind() == constant.Unknown {
			return nil, fmt.Errorf("%w: unary %s", ErrEnum, e.Op)
		}
		return out, nil
	case *ast.BinaryExpr:
		x, err := eval(e.X, iota, env)
		if err != nil {
			return nil, err
		}
		y, err := eval(e.Y, iota, env)
		if err != nil {
			return nil, err
		}
		if e.Op == token.SHL || e.Op == token.SHR {
			n, exact := constant.Uint64Val(constant.ToInt(y))
			if !exact || n > 4096 {
				return nil, fmt.Errorf("%w: shift", ErrEnum)
			}
			out := constant.Shift(x, e.Op, uint(n))
			if out.Kind() == constant.Unknown {
				return nil, fmt.Errorf("%w: shift", ErrEnum)
			}
			return out, nil
		}
		out := constant.BinaryOp(x, e.Op, y)
		if out.Kind() == constant.Unknown {
			return nil, fmt.Errorf("%w: %s", ErrEnum, e.Op)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("%w: expression", ErrEnum)
	}
}

func leadingBuild(src string) string {
	var lines []string
	for _, line := range strings.Split(src, "\n") {
		trim := strings.TrimSpace(line)
		if trim == "" {
			if len(lines) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(trim, "//go:build") || strings.HasPrefix(trim, "// +build") {
			lines = append(lines, trim)
			continue
		}
		break
	}
	return strings.Join(lines, "\n")
}
