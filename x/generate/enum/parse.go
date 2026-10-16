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
	members, exprs, err := constMembers(file, marks)
	if err != nil {
		return fileSpec{}, err
	}
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

func constMembers(file *ast.File, marks map[string]mark) (map[string][]member, map[string]ast.Expr, error) {
	env := map[string]constant.Value{}
	out := map[string][]member{}
	exprs := map[string]ast.Expr{}
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
				return nil, nil, fmt.Errorf("%w: const list", ErrEnum)
			}
			iotaVal := constant.MakeInt64(int64(index))
			for i, name := range vs.Names {
				val, err := eval(values[i], iotaVal, env)
				if err != nil {
					return nil, nil, err
				}
				if name.Name != "_" {
					if _, exists := env[name.Name]; exists {
						return nil, nil, fmt.Errorf("%w: %s declared twice", ErrEnum, name.Name)
					}
					env[name.Name] = val
				}
				itemType := typeName
				if itemType == "" {
					if converted, ok := convertedType(values[i]); ok {
						if part, marked := marks[converted]; marked && part.directive.composeLeft != "" {
							itemType = converted
						}
					}
				}
				mark, wanted := marks[itemType]
				if !wanted || name.Name == "_" {
					continue
				}
				for _, prev := range seenVal[itemType] {
					if constant.Compare(prev, token.EQL, val) {
						return nil, nil, fmt.Errorf("%w: %s repeats a value", ErrEnum, name.Name)
					}
				}
				seenVal[itemType] = append(seenVal[itemType], val)
				item, err := memberOf(itemType, name.Name, val, mark, vs.Comment)
				if err != nil {
					return nil, nil, err
				}
				exprs[name.Name] = values[i]
				out[itemType] = append(out[itemType], item)
			}
		}
	}
	return out, exprs, nil
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
		if !spec.stringKind {
			return fmt.Errorf("%w: %s compose requires a string", ErrEnum, spec.name)
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
			leftName, rightName, ok := compositionParts(exprs[item.name], spec.name)
			if !ok {
				return fmt.Errorf("%w: %s is not a %s/%s composition", ErrEnum, item.name, spec.composeLeft, spec.composeRight)
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
			tokenText := leftMember.token + "/" + rightMember.token
			if item.token != tokenText {
				return fmt.Errorf("%w: %s text %s", ErrEnum, item.name, item.token)
			}
			item.token = tokenText
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

func compositionParts(expr ast.Expr, typeName string) (string, string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return "", "", false
	}
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || fun.Name != typeName {
		return "", "", false
	}
	parts := flattenAdd(call.Args[0])
	if len(parts) != 3 {
		return "", "", false
	}
	left, ok := stringConstIdent(parts[0])
	if !ok {
		return "", "", false
	}
	lit, ok := parts[1].(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING || lit.Value != `"/"` {
		return "", "", false
	}
	right, ok := stringConstIdent(parts[2])
	if !ok {
		return "", "", false
	}
	return left, right, true
}

func flattenAdd(expr ast.Expr) []ast.Expr {
	if paren, ok := expr.(*ast.ParenExpr); ok {
		return flattenAdd(paren.X)
	}
	bin, ok := expr.(*ast.BinaryExpr)
	if !ok || bin.Op != token.ADD {
		return []ast.Expr{expr}
	}
	return append(flattenAdd(bin.X), flattenAdd(bin.Y)...)
}

func stringConstIdent(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return "", false
	}
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || fun.Name != "string" {
		return "", false
	}
	id, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, true
}

func convertedType(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 || call.Ellipsis.IsValid() {
		return "", false
	}
	fun, ok := call.Fun.(*ast.Ident)
	if !ok || !plainIdent(fun.Name) {
		return "", false
	}
	return fun.Name, true
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
	case *ast.CallExpr:
		fun, ok := e.Fun.(*ast.Ident)
		if !ok || len(e.Args) != 1 || e.Ellipsis.IsValid() || (fun.Name != "string" && !plainIdent(fun.Name)) {
			return nil, fmt.Errorf("%w: expression", ErrEnum)
		}
		val, err := eval(e.Args[0], iota, env)
		if err != nil {
			return nil, err
		}
		if val.Kind() != constant.String {
			return nil, fmt.Errorf("%w: expression", ErrEnum)
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
