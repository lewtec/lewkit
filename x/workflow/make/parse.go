package make

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lewtec/leaven-tree-sitter/grammar/make"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	_ "github.com/lewtec/lewkit/x/driver/treesitter/leaven"
	"github.com/lewtec/lewkit/x/workflow"
)

func (f *File) includeFile(ctx context.Context, path string, optional bool, goal, from string, line int) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if f.included[abs] {
		return nil
	}
	data, err := os.ReadFile(abs)
	if err != nil {
		if os.IsNotExist(err) && from != "" {
			// GNU make keeps reading, then remakes a missing include.
			for _, old := range f.missing {
				if old.abs == abs {
					return nil
				}
			}
			f.missing = append(f.missing, missedInclude{
				name:     goal,
				abs:      abs,
				from:     from,
				line:     line,
				optional: optional,
			})
			return nil
		}
		return err
	}
	f.included[abs] = true
	f.pushMakefile(abs)
	return f.parseSrc(ctx, abs, string(data))
}

func (f *File) pushMakefile(path string) {
	const key = "MAKEFILE_LIST"
	cur := ""
	if v := f.vars[key]; v != nil {
		cur = v.value
	}
	if cur != "" {
		cur += " "
	}
	f.vars[key] = &variable{value: cur + path, simple: true, origin: originAutomatic}
}

func (f *File) parseSrc(ctx context.Context, path, src string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	src = prepareSrc(src)
	tree, err := treesitter.Parse(ctx, "make", []byte(src))
	if err != nil {
		return syntax(path, 1, err.Error())
	}
	root := tree.RootNode()
	if err := tree.Parsed(); err != nil {
		return syntax(path, lineAt(src, errorNode(root)), "syntax error")
	}
	if root == nil || root.IsNull() {
		return nil
	}
	for i := uint32(0); i < root.NamedChildCount(); i++ {
		if err := f.walkNode(ctx, path, src, root.NamedChild(i), originFile); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) walkNode(ctx context.Context, path, src string, n treesitter.Node, origin origin) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if n == nil || n.IsNull() || !n.IsNamed() {
		return nil
	}
	switch n.Type() {
	case "comment":
		return nil
	case "rule":
		return f.walkRule(path, src, n)
	case "variable_assignment", "shell_assignment", "VPATH_assignment", "RECIPEPREFIX_assignment":
		return f.walkAssign(path, src, n, origin, false)
	case "define_directive":
		return f.walkDefine(path, src, n, origin, false)
	case "include_directive":
		return f.walkInclude(ctx, path, src, n)
	case "export_directive":
		return f.walkExport(path, src, n, origin)
	case "unexport_directive":
		return f.walkUnexport(src, n)
	case "override_directive":
		return f.walkOverride(ctx, path, src, n)
	case "conditional":
		return f.walkConditional(ctx, path, src, n, origin)
	case "function_call", "shell_function":
		_, err := f.expand(strings.TrimSpace(nodeText(src, n)), nil)
		return err
	default:
		if n.IsError() || n.Type() == "ERROR" {
			return syntax(path, lineAt(src, n), "syntax error")
		}
		return syntax(path, lineAt(src, n), "unsupported "+n.Type())
	}
}

func (f *File) walkConditional(ctx context.Context, path, src string, n treesitter.Node, origin origin) error {
	taken := false
	active := false
	seen := false
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) == "condition" {
			c := n.Child(i)
			ok, err := f.condTrue(src, c)
			if err != nil {
				return syntax(path, lineAt(src, c), err.Error())
			}
			active, taken, seen = ok, ok, true
			continue
		}
		c := n.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		switch c.Type() {
		case "elsif_directive":
			if !seen {
				return syntax(path, lineAt(src, c), "else without if")
			}
			if taken {
				active = false
				continue
			}
			ok, err := f.condTrue(src, field(c, "condition"))
			if err != nil {
				return syntax(path, lineAt(src, c), err.Error())
			}
			active = ok
			if ok {
				taken = true
			}
			if active {
				if err := f.walkBlock(ctx, path, src, c, origin); err != nil {
					return err
				}
			}
		case "else_directive":
			if !seen {
				return syntax(path, lineAt(src, c), "else without if")
			}
			active = !taken
			taken = true
			if active {
				if err := f.walkBlock(ctx, path, src, c, origin); err != nil {
					return err
				}
			}
		default:
			if active {
				if err := f.walkNode(ctx, path, src, c, origin); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *File) walkBlock(ctx context.Context, path, src string, n treesitter.Node, origin origin) error {
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) == "condition" {
			continue
		}
		c := n.Child(i)
		if c == nil || c.IsNull() || !c.IsNamed() {
			continue
		}
		if err := f.walkNode(ctx, path, src, c, origin); err != nil {
			return err
		}
	}
	return nil
}

func (f *File) condTrue(src string, n treesitter.Node) (bool, error) {
	if n == nil || n.IsNull() {
		return false, errString("bad conditional")
	}
	switch n.Type() {
	case "ifeq_directive", "ifneq_directive":
		left, err := f.expand(argText(src, field(n, "arg0")), nil)
		if err != nil {
			return false, err
		}
		right, err := f.expand(argText(src, field(n, "arg1")), nil)
		if err != nil {
			return false, err
		}
		same := left == right
		if n.Type() == "ifneq_directive" {
			return !same, nil
		}
		return same, nil
	case "ifdef_directive", "ifndef_directive":
		name := strings.TrimSpace(nodeText(src, field(n, "variable")))
		// GNU make tests the stored text, not the expanded value.
		// An empty := is false. A recursive value $(unset) is still true.
		v := f.vars[name]
		set := v != nil && v.value != ""
		if n.Type() == "ifndef_directive" {
			return !set, nil
		}
		return set, nil
	default:
		return false, errString("bad conditional")
	}
}

func (f *File) walkAssign(path, src string, n treesitter.Node, origin origin, export bool) error {
	raw := strings.TrimSpace(nodeText(src, field(n, "name")))
	if raw == targetAssignName {
		return f.addTargetAssign(path, src, n, origin)
	}
	if raw == rawAssignName {
		return f.addRawAssign(path, src, n, origin, export)
	}
	name, err := f.expand(decodeRefs(raw), nil)
	if err != nil {
		return syntax(path, lineAt(src, n), err.Error())
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return syntax(path, lineAt(src, n), "bad assignment")
	}
	op := strings.TrimSpace(nodeText(src, field(n, "operator")))
	if op == "" {
		if n.Type() == "shell_assignment" {
			op = "!="
		} else {
			op = "="
		}
	}
	value := strings.TrimSpace(nodeText(src, field(n, "value")))
	if err := f.assign(name, value, op, origin); err != nil {
		return err
	}
	if export {
		f.exported[name] = true
		delete(f.unexported, name)
	}
	return nil
}

func (f *File) addRawAssign(path, src string, n treesitter.Node, origin origin, export bool) error {
	name, op, value, exp, over, ok := decodeRaw(nodeText(src, field(n, "value")))
	if !ok || op == "" {
		return syntax(path, lineAt(src, n), "bad assignment")
	}
	expanded, err := f.expand(decodeRefs(name), nil)
	if err != nil {
		return syntax(path, lineAt(src, n), err.Error())
	}
	expanded = strings.TrimSpace(expanded)
	if expanded == "" {
		return syntax(path, lineAt(src, n), "bad assignment")
	}
	if over && origin < originOverride {
		origin = originOverride
	}
	if err := f.assign(expanded, value, op, origin); err != nil {
		return err
	}
	if export || exp {
		f.exported[expanded] = true
		delete(f.unexported, expanded)
	}
	return nil
}

func (f *File) addTargetAssign(path, src string, n treesitter.Node, origin origin) error {
	rec, ok := decodeTarget(nodeText(src, field(n, "value")))
	if !ok {
		return syntax(path, lineAt(src, n), "bad target assignment")
	}
	rawTargets, err := f.expand(decodeRefs(rec.targets), nil)
	if err != nil {
		return syntax(path, lineAt(src, n), err.Error())
	}
	targets := strings.Fields(rawTargets)
	name, err := f.expand(decodeRefs(rec.name), nil)
	if err != nil {
		return syntax(path, lineAt(src, n), err.Error())
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return syntax(path, lineAt(src, n), "bad target assignment")
	}
	// GNU make ignores a target-specific assignment whose target list expands empty.
	if len(targets) == 0 {
		return nil
	}
	if rec.over && origin < originOverride {
		origin = originOverride
	}
	tv := targetVar{
		targets: targets,
		name:    name,
		op:      rec.op,
		value:   rec.value,
		export:  rec.export,
		private: rec.private,
		origin:  origin,
	}
	if kindOf(rec.op) == kindSimple {
		folded, err := f.foldTarget(targets[0], rec.value)
		if err != nil {
			return syntax(path, lineAt(src, n), err.Error())
		}
		tv.value = folded
		tv.folded = true
	}
	f.targetVars = append(f.targetVars, tv)
	return nil
}

func (f *File) foldTarget(target, value string) (string, error) {
	var out string
	err := f.withTargetVars(target, func() error {
		var expandErr error
		out, expandErr = f.expand(value, nil)
		return expandErr
	})
	return out, err
}

func (f *File) walkDefine(path, src string, n treesitter.Node, origin origin, export bool) error {
	raw := strings.TrimSpace(nodeText(src, field(n, "name")))
	name, err := f.expand(decodeRefs(raw), nil)
	if err != nil {
		return syntax(path, lineAt(src, n), err.Error())
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return syntax(path, lineAt(src, n), "bad assignment")
	}
	op := strings.TrimSpace(nodeText(src, field(n, "operator")))
	if op == "" {
		op = "="
	}
	value := strings.TrimRight(nodeText(src, field(n, "value")), "\n")
	if err := f.assign(name, value, op, origin); err != nil {
		return err
	}
	if export {
		f.exported[name] = true
		delete(f.unexported, name)
	}
	return nil
}

func (f *File) walkInclude(ctx context.Context, path, src string, n treesitter.Node) error {
	optional := includeOptional(src, n)
	names, err := f.expandAll(texts(src, field(n, "filenames")))
	if err != nil {
		return err
	}
	base := filepath.Dir(path)
	line := lineAt(src, n)
	for _, item := range names {
		if item == "" {
			continue
		}
		if err := f.includeFile(ctx, workflow.Join(base, item), optional, item, path, line); err != nil {
			return err
		}
	}
	return nil
}

func includeOptional(src string, n treesitter.Node) bool {
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() || c.IsNamed() {
			continue
		}
		switch strings.TrimSpace(nodeText(src, c)) {
		case "-include", "sinclude":
			return true
		}
	}
	return false
}

func (f *File) walkExport(path, src string, n treesitter.Node, origin origin) error {
	saw := false
	for _, c := range namedChildren(n) {
		switch c.Type() {
		case "variable_assignment", "shell_assignment", "VPATH_assignment", "RECIPEPREFIX_assignment":
			saw = true
			if err := f.walkAssign(path, src, c, origin, true); err != nil {
				return err
			}
		case "define_directive":
			saw = true
			if err := f.walkDefine(path, src, c, origin, true); err != nil {
				return err
			}
		}
	}
	if saw {
		return nil
	}
	for _, name := range texts(src, field(n, "variables")) {
		if !ident(name) {
			continue
		}
		f.exported[name] = true
		delete(f.unexported, name)
	}
	return nil
}

func (f *File) walkUnexport(src string, n treesitter.Node) error {
	for _, name := range texts(src, field(n, "variables")) {
		if !ident(name) {
			continue
		}
		delete(f.exported, name)
		f.unexported[name] = true
	}
	return nil
}

func (f *File) walkOverride(ctx context.Context, path, src string, n treesitter.Node) error {
	for _, c := range namedChildren(n) {
		switch c.Type() {
		case "variable_assignment", "shell_assignment", "VPATH_assignment", "RECIPEPREFIX_assignment":
			if err := f.walkAssign(path, src, c, originOverride, false); err != nil {
				return err
			}
		case "define_directive":
			if err := f.walkDefine(path, src, c, originOverride, false); err != nil {
				return err
			}
		case "export_directive":
			if err := f.walkExport(path, src, c, originOverride); err != nil {
				return err
			}
		default:
			if err := f.walkNode(ctx, path, src, c, originOverride); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *File) walkRule(path, src string, n treesitter.Node) error {
	lines := recipeLines(src, childType(n, "recipe"))
	if pat := field(n, "target"); pat != nil {
		return f.walkStatic(path, src, n, pat, lines)
	}
	rawTargets := texts(src, childType(n, "targets"))
	names, err := f.expandAll(rawTargets)
	if err != nil {
		return err
	}
	pre, err := f.expandAll(texts(src, field(n, "normal")))
	if err != nil {
		return err
	}
	ord, err := f.expandAll(texts(src, field(n, "order_only")))
	if err != nil {
		return err
	}
	if patternRule(rawTargets) {
		body := &rule{
			prereqs: append([]string{}, pre...),
			order:   append([]string{}, ord...),
			recipe:  append([]string{}, lines...),
		}
		for _, name := range names {
			f.patterns = append(f.patterns, pattern{target: name, body: body})
		}
		return nil
	}
	for _, name := range names {
		f.noteSpecial(name, pre)
		if specialTarget(name) {
			continue
		}
		setRecipe(f.touchExplicit(name, pre, ord, ""), lines)
	}
	return nil
}

func (f *File) walkStatic(path, src string, n, pat treesitter.Node, lines []string) error {
	names, err := f.expandAll(texts(src, childType(n, "targets")))
	if err != nil {
		return err
	}
	pats, err := f.expandAll(texts(src, pat))
	if err != nil {
		return err
	}
	if len(pats) == 0 {
		return syntax(path, lineAt(src, n), "static pattern needs a target pattern")
	}
	pre, err := f.expandAll(texts(src, field(n, "prerequisite")))
	if err != nil {
		return err
	}
	ord, err := f.expandAll(texts(src, field(n, "order_only")))
	if err != nil {
		return err
	}
	for _, name := range names {
		stem, ok := stemOf(pats[0], name)
		if !ok {
			return syntax(path, lineAt(src, n), "target pattern does not match "+name)
		}
		setRecipe(f.touchExplicit(name, applyStem(pre, stem), applyStem(ord, stem), stem), lines)
	}
	return nil
}

func patternRule(targets []string) bool {
	for _, target := range targets {
		if strings.Contains(target, "%") {
			return true
		}
	}
	return false
}

func recipeLines(src string, n treesitter.Node) []string {
	var lines []string
	for _, c := range namedChildren(n) {
		if c.Type() != "recipe_line" {
			continue
		}
		lines = append(lines, strings.TrimRight(nodeText(src, c), "\n"))
	}
	return lines
}

func setRecipe(r *rule, lines []string) {
	if r == nil || len(lines) == 0 {
		return
	}
	r.recipe = append([]string{}, lines...)
}

func (f *File) expandAll(parts []string) ([]string, error) {
	var out []string
	for _, part := range parts {
		expanded, err := f.expand(part, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, strings.Fields(expanded)...)
	}
	return out, nil
}

func (f *File) applyAssign(line string, origin origin, path string, no int) error {
	name, op, value, override, ok := splitAssign(line)
	if !ok || name == "" {
		if no == 0 {
			return syntax(path, 1, "bad assignment "+line)
		}
		return syntax(path, no, "bad assignment")
	}
	if override && origin < originOverride {
		origin = originOverride
	}
	return f.assign(name, value, op, origin)
}

func (f *File) touchExplicit(name string, pre, ord []string, stem string) *rule {
	got := f.explicit[name]
	if got == nil {
		got = &rule{}
		f.explicit[name] = got
		if f.first == "" && !strings.HasPrefix(name, ".") {
			f.first = name
		}
	}
	got.prereqs = append(got.prereqs, pre...)
	got.order = append(got.order, ord...)
	if stem != "" {
		got.stem = stem
	}
	return got
}

func (f *File) noteSpecial(name string, prereqs []string) {
	switch name {
	case ".PHONY":
		for _, p := range prereqs {
			f.phonies[p] = true
		}
	case ".EXPORT_ALL_VARIABLES":
		f.exportAll = true
	case ".ONESHELL":
		f.oneshell = true
	case ".DEFAULT_GOAL":
		if len(prereqs) > 0 {
			f.vars[".DEFAULT_GOAL"] = &variable{value: strings.Join(prereqs, " "), simple: true, origin: originFile}
		}
	}
}

func specialTarget(name string) bool {
	switch name {
	case ".PHONY", ".SUFFIXES", ".DEFAULT", ".IGNORE", ".SILENT",
		".DELETE_ON_ERROR", ".PRECIOUS", ".INTERMEDIATE", ".SECONDARY",
		".NOTPARALLEL", ".EXPORT_ALL_VARIABLES", ".ONESHELL", ".POSIX",
		".MAKE", ".WAIT", ".DEFAULT_GOAL":
		return true
	default:
		return false
	}
}

func namedChildren(n treesitter.Node) []treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	out := make([]treesitter.Node, 0, n.NamedChildCount())
	for i := uint32(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if c == nil || c.IsNull() {
			continue
		}
		out = append(out, c)
	}
	return out
}

func texts(src string, n treesitter.Node) []string {
	var out []string
	for _, c := range namedChildren(n) {
		text := strings.TrimSpace(nodeText(src, c))
		if text != "" {
			out = append(out, text)
		}
	}
	return out
}

func field(n treesitter.Node, name string) treesitter.Node {
	if n == nil || n.IsNull() {
		return nil
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		if n.FieldNameForChild(i) != name {
			continue
		}
		c := n.Child(i)
		if c != nil && !c.IsNull() {
			return c
		}
	}
	return nil
}

func childType(n treesitter.Node, typ string) treesitter.Node {
	for _, c := range namedChildren(n) {
		if c.Type() == typ {
			return c
		}
	}
	return nil
}

func nodeText(src string, n treesitter.Node) string {
	if n == nil || n.IsNull() {
		return ""
	}
	start, end := int(n.StartByte()), int(n.EndByte())
	if start < 0 || end < start || start > len(src) {
		return ""
	}
	if end > len(src) {
		end = len(src)
	}
	return src[start:end]
}

func argText(src string, n treesitter.Node) string {
	raw := nodeText(src, n)
	if n == nil || n.IsNull() || n.Type() != "string" {
		return raw
	}
	raw = strings.TrimSpace(raw)
	if len(raw) >= 2 {
		q := raw[0]
		if (q == '"' || q == '\'') && raw[len(raw)-1] == q {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

func lineAt(src string, n treesitter.Node) int {
	if n == nil || n.IsNull() {
		return 1
	}
	off := int(n.StartByte())
	if off < 0 {
		return 1
	}
	if off > len(src) {
		off = len(src)
	}
	return 1 + strings.Count(src[:off], "\n")
}

func errorNode(n treesitter.Node) treesitter.Node {
	if n == nil || n.IsNull() || !n.HasError() {
		return n
	}
	for i := uint32(0); i < n.ChildCount(); i++ {
		c := n.Child(i)
		if c == nil || c.IsNull() || !c.HasError() {
			continue
		}
		return errorNode(c)
	}
	return n
}

type errString string

func (e errString) Error() string { return string(e) }
