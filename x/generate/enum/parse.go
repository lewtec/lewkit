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
	name         string
	stringKind   bool
	fold         bool
	empty        string
	invalid      string
	hasInvalid   bool
	composeLeft  string
	composeRight string
	members      []member
}

type member struct {
	name    string
	token   string
	aliases []string
}

type directive struct {
	fold         bool
	empty        string
	invalid      string
	hasInvalid   bool
	composeLeft  string
	composeRight string
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
	found := &pairFound{
		marks:    marks,
		members:  members,
		exprs:    map[string]ast.Expr{},
		declared: map[string]bool{},
	}
	if err := found.collect(file); err != nil {
		return fileSpec{}, err
	}
	exprs := found.exprs
	var enums []enumSpec
	for _, name := range markOrder(file, marks) {
		mark := marks[name]
		got := members[name]
		if len(got) == 0 {
			return fileSpec{}, fmt.Errorf("%w: %s has no members", ErrEnum, name)
		}
		enums = append(enums, enumSpec{
			name:         name,
			stringKind:   mark.stringKind,
			fold:         mark.directive.fold,
			empty:        mark.directive.empty,
			invalid:      mark.directive.invalid,
			hasInvalid:   mark.directive.hasInvalid,
			composeLeft:  mark.directive.composeLeft,
			composeRight: mark.directive.composeRight,
			members:      got,
		})
	}
	if err := applyCompose(enums, exprs); err != nil {
		return fileSpec{}, err
	}
	for i := range enums {
		if err := checkMembers(enums[i].name, marks[enums[i].name], enums[i].members); err != nil {
			return fileSpec{}, err
		}
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
			var stringKind bool
			if dir.composeLeft != "" {
				if err := pairType(ts.Name.Name, ts.Type, dir); err != nil {
					return nil, err
				}
			} else {
				kind, ok := underlying(ts.Type)
				if !ok {
					return nil, fmt.Errorf("%w: %s is not an integer or string", ErrEnum, ts.Name.Name)
				}
				stringKind = kind == "string"
			}
			if _, dup := out[ts.Name.Name]; dup {
				return nil, fmt.Errorf("%w: %s declared twice", ErrEnum, ts.Name.Name)
			}
			out[ts.Name.Name] = mark{stringKind: stringKind, directive: dir}
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
	for _, field := range fields[1:] {
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
		case strings.HasPrefix(field, "compose="):
			raw := strings.TrimPrefix(field, "compose=")
			left, right, ok := strings.Cut(raw, "/")
			if !ok || !plainIdent(left) || !plainIdent(right) || left == right {
				return directive{}, fmt.Errorf("%w: compose option", ErrEnum)
			}
			dir.composeLeft = left
			dir.composeRight = right
		default:
			return directive{}, fmt.Errorf("%w: %s", ErrEnum, field)
		}
	}
	return dir, nil
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
				if mark, ok := marks[typeName]; ok && mark.directive.composeLeft != "" {
					return nil, fmt.Errorf("%w: %s compose var", ErrEnum, name.Name)
				}
				if lit, ok := values[i].(*ast.CompositeLit); ok && lit.Type != nil {
					if id, ok := lit.Type.(*ast.Ident); ok {
						if mark, marked := marks[id.Name]; marked && mark.directive.composeLeft != "" {
							return nil, fmt.Errorf("%w: %s compose var", ErrEnum, name.Name)
						}
					}
				}
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
	return member{name: name, token: tokenText, aliases: extra}, nil
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
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '+' || r == '.' || r == '&' || r == '-' || r == '/' {
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

func applyCompose(enums []enumSpec, exprs map[string]ast.Expr) error {
	byName := make(map[string]enumSpec, len(enums))
	for _, spec := range enums {
		byName[spec.name] = spec
	}
	for i := range enums {
		spec := &enums[i]
		if spec.composeLeft == "" {
			continue
		}
		if generatedMethod(spec.composeLeft) || generatedMethod(spec.composeRight) {
			return fmt.Errorf("%w: %s compose method", ErrEnum, spec.name)
		}
		left, ok := byName[spec.composeLeft]
		if !ok {
			return fmt.Errorf("%w: %s compose %s", ErrEnum, spec.name, spec.composeLeft)
		}
		right, ok := byName[spec.composeRight]
		if !ok {
			return fmt.Errorf("%w: %s compose %s", ErrEnum, spec.name, spec.composeRight)
		}
		if !left.stringKind || !right.stringKind || left.composeLeft != "" || right.composeLeft != "" {
			return fmt.Errorf("%w: %s compose part", ErrEnum, spec.name)
		}
		leftByName := membersByName(left.members)
		rightByName := membersByName(right.members)
		for j := range spec.members {
			item := &spec.members[j]
			leftName, rightName, ok := compositionParts(exprs[item.name], spec.composeLeft, spec.composeRight)
			if !ok {
				return fmt.Errorf("%w: %s is not a %s/%s pair", ErrEnum, item.name, spec.composeLeft, spec.composeRight)
			}
			leftMember, ok := leftByName[leftName]
			if !ok {
				return fmt.Errorf("%w: %s part %s", ErrEnum, item.name, leftName)
			}
			rightMember, ok := rightByName[rightName]
			if !ok {
				return fmt.Errorf("%w: %s part %s", ErrEnum, item.name, rightName)
			}
			if err := separatorNames(item.name, leftMember, rightMember); err != nil {
				return err
			}
			item.token = leftMember.token + "/" + rightMember.token
			item.aliases = composeAliases(leftMember, rightMember, item.aliases)
		}
	}
	return nil
}

func membersByName(members []member) map[string]member {
	out := make(map[string]member, len(members))
	for _, item := range members {
		out[item.name] = item
	}
	return out
}

func separatorNames(owner string, left, right member) error {
	for _, name := range partNames(left) {
		if strings.Contains(name, "/") {
			return fmt.Errorf("%w: %s separator %q", ErrEnum, owner, name)
		}
	}
	for _, name := range partNames(right) {
		if strings.Contains(name, "/") {
			return fmt.Errorf("%w: %s separator %q", ErrEnum, owner, name)
		}
	}
	return nil
}

func partNames(item member) []string {
	names := make([]string, 0, 1+len(item.aliases))
	names = append(names, item.token)
	names = append(names, item.aliases...)
	return names
}

func composeAliases(left, right member, extra []string) []string {
	seen := map[string]bool{left.token + "/" + right.token: true}
	var out []string
	add := func(text string) {
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		out = append(out, text)
	}
	for _, leftName := range partNames(left) {
		for _, rightName := range partNames(right) {
			add(leftName + "/" + rightName)
			add(leftName + "_" + rightName)
		}
	}
	for _, text := range extra {
		add(text)
	}
	return out
}

func pairType(name string, expr ast.Expr, dir directive) error {
	st, ok := expr.(*ast.StructType)
	if !ok || st.Fields == nil || len(st.Fields.List) != 2 {
		return fmt.Errorf("%w: %s compose pair", ErrEnum, name)
	}
	want := []string{dir.composeLeft, dir.composeRight}
	for i, field := range st.Fields.List {
		id, ok := field.Type.(*ast.Ident)
		if !ok || field.Tag != nil || len(field.Names) != 1 || field.Names[0].Name != want[i] || id.Name != want[i] {
			return fmt.Errorf("%w: %s compose pair", ErrEnum, name)
		}
	}
	return nil
}

type pairFound struct {
	marks    map[string]mark
	members  map[string][]member
	exprs    map[string]ast.Expr
	declared map[string]bool
}

func (found *pairFound) collect(file *ast.File) error {
	for _, items := range found.members {
		for _, item := range items {
			found.declared[item.name] = true
		}
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			if err := found.add(spec.(*ast.ValueSpec)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (found *pairFound) add(vs *ast.ValueSpec) error {
	explicit := ""
	if id, ok := vs.Type.(*ast.Ident); ok {
		explicit = id.Name
	}
	if len(vs.Values) != len(vs.Names) {
		if mark, ok := found.marks[explicit]; ok && mark.directive.composeLeft != "" {
			return fmt.Errorf("%w: %s compose var", ErrEnum, explicit)
		}
		return nil
	}
	for i, name := range vs.Names {
		if name.Name == "_" {
			continue
		}
		lit, isLit := vs.Values[i].(*ast.CompositeLit)
		itemType := explicit
		litType := ""
		if isLit {
			if id, ok := lit.Type.(*ast.Ident); ok {
				litType = id.Name
			}
		}
		if itemType == "" {
			itemType = litType
		}
		mark, wanted := found.marks[itemType]
		if !wanted || mark.directive.composeLeft == "" {
			continue
		}
		if !isLit || (lit.Type != nil && litType != itemType) {
			return fmt.Errorf("%w: %s is not a %s/%s pair", ErrEnum, name.Name, mark.directive.composeLeft, mark.directive.composeRight)
		}
		if found.declared[name.Name] {
			return fmt.Errorf("%w: %s declared twice", ErrEnum, name.Name)
		}
		found.declared[name.Name] = true
		found.members[itemType] = append(found.members[itemType], member{
			name:    name.Name,
			aliases: aliasComment(vs.Comment),
		})
		found.exprs[name.Name] = lit
	}
	return nil
}

func compositionParts(expr ast.Expr, leftField, rightField string) (string, string, bool) {
	lit, ok := expr.(*ast.CompositeLit)
	if !ok || len(lit.Elts) != 2 {
		return "", "", false
	}
	if _, keyed := lit.Elts[0].(*ast.KeyValueExpr); keyed {
		return keyedParts(lit.Elts, leftField, rightField)
	}
	left, ok := valueIdent(lit.Elts[0])
	if !ok {
		return "", "", false
	}
	right, ok := valueIdent(lit.Elts[1])
	if !ok {
		return "", "", false
	}
	return left, right, true
}

func keyedParts(elts []ast.Expr, leftField, rightField string) (string, string, bool) {
	found := map[string]string{}
	for _, elt := range elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return "", "", false
		}
		key, ok := kv.Key.(*ast.Ident)
		if !ok {
			return "", "", false
		}
		value, ok := valueIdent(kv.Value)
		if !ok {
			return "", "", false
		}
		if _, dup := found[key.Name]; dup {
			return "", "", false
		}
		found[key.Name] = value
	}
	left, ok := found[leftField]
	if !ok {
		return "", "", false
	}
	right, ok := found[rightField]
	if !ok || len(found) != 2 {
		return "", "", false
	}
	return left, right, true
}

func valueIdent(expr ast.Expr) (string, bool) {
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return valueIdent(paren.X)
	}
	id, ok := expr.(*ast.Ident)
	if !ok || !plainIdent(id.Name) {
		return "", false
	}
	return id.Name, true
}

func plainIdent(name string) bool {
	return token.IsIdentifier(name) && !token.IsKeyword(name)
}

func generatedMethod(name string) bool {
	switch name {
	case "String", "Values", "IsValid", "Parse", "ArgChoices", "MarshalText", "UnmarshalText":
		return true
	default:
		return false
	}
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
