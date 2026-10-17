package makefile

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Production]
	langErr  error
)

func language() (*xparse.Model[Term, Production], error) {
	langOnce.Do(func() {
		lang, langErr = grammar()
	})
	return lang, langErr
}

func grammar() (*xparse.Model[Term, Production], error) {
	var syntax xparse.Syntax[Term, Production]
	assign := syntax.Term(TermAssign, nil)
	rule := syntax.Term(TermRule, nil)
	recipe := syntax.Term(TermRecipe, nil)
	ifTok := syntax.Term(TermIf, nil)
	elseTok := syntax.Term(TermElse, nil)
	endif := syntax.Term(TermEndif, nil)
	defineTok := syntax.Term(TermDefine, nil)
	body := syntax.Term(TermBody, nil)
	endef := syntax.Term(TermEndef, nil)
	include := syntax.Term(TermInclude, nil)
	vpath := syntax.Term(TermVPath, nil)
	export := syntax.Term(TermExport, nil)
	undefine := syntax.Term(TermUndefine, nil)
	unknown := syntax.Term(TermUnknown, nil)

	statement := syntax.Eq(ProductionStatement, syntax.Alt(
		assign,
		rule,
		recipe,
		include,
		vpath,
		export,
		undefine,
		unknown,
		syntax.Ref(ProductionCondition),
		syntax.Ref(ProductionDefine),
	))
	elseBranch := syntax.Eq(ProductionElse, syntax.Seq(
		elseTok,
		syntax.Repeat(syntax.Ref(ProductionStatement)),
	))
	condition := syntax.Eq(ProductionCondition, syntax.Seq(
		ifTok,
		syntax.Repeat(syntax.Ref(ProductionStatement)),
		syntax.Repeat(syntax.Ref(ProductionElse)),
		endif,
	))
	// endef is optional so a missing closer is reported on the define line.
	defineRule := syntax.Eq(ProductionDefine, syntax.Seq(
		defineTok,
		syntax.Repeat(body),
		syntax.Opt(endef),
	))
	file := syntax.Eq(ProductionFile, syntax.Repeat(syntax.Ref(ProductionStatement)))
	return syntax.Compile("makefile", syntax.Compose(
		syntax.Scan(scan),
		file,
		statement,
		elseBranch,
		condition,
		defineRule,
		syntax.Start(ProductionFile),
	))
}

type adder interface {
	add(Stmt)
	last() Stmt
}

func parse(text string) (*File, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(text))
	if err != nil {
		return nil, parseErr(text, err)
	}
	return lower(text, root)
}

func parseErr(text string, err error) error {
	var pe *xparse.Error
	if !errors.As(err, &pe) {
		return syn(1, "syntax")
	}
	if pe.Offset >= len(text) {
		for _, want := range pe.Want {
			if want == TermEndif.String() {
				return syn(pe.Line, "missing endif")
			}
			if want == TermEndef.String() {
				return syn(pe.Line, "missing endef")
			}
		}
		return syn(pe.Line, "syntax")
	}
	at := pe.Offset
	if at < 0 {
		at = 0
	}
	src := []byte(text)
	end, _, ok := readLogical(src, at)
	if !ok {
		return syn(pe.Line, "unterminated continuation")
	}
	raw := string(src[at:end])
	collapsed := collapse(raw)
	if !strings.HasPrefix(collapsed, "\t") {
		word, rest := cutWord(trimLeft(collapsed))
		if word == "endef" && trimSpace(removeComments(rest)) != "" {
			return syn(pe.Line, "extraneous text after endef")
		}
	}
	switch classOf(cookedLine(raw)) {
	case kindElse:
		return syn(pe.Line, "extraneous else")
	case kindEndif:
		_, rest := cutWord(cookedLine(raw))
		if trimSpace(rest) != "" {
			return syn(pe.Line, "extraneous text after endif")
		}
		return syn(pe.Line, "extraneous endif")
	}
	return syn(pe.Line, "syntax")
}

func lower(text string, root xparse.Syn[Term, Production]) (*File, error) {
	file := New()
	if err := appendStmts(text, root, file); err != nil {
		return nil, err
	}
	return file, nil
}

func appendStmts(text string, parent xparse.Syn[Term, Production], dst adder) error {
	for i := 0; i < parent.ChildCount(); i++ {
		if err := appendStmt(text, parent.Child(i), dst); err != nil {
			return err
		}
	}
	return nil
}

func appendStmt(text string, node xparse.Syn[Term, Production], dst adder) error {
	if rule, ok := node.Rule(); ok && rule == ProductionStatement {
		if node.ChildCount() != 1 {
			return syn(lineAt(text, node.Start()), "syntax")
		}
		node = node.Child(0)
	}
	if rule, ok := node.Rule(); ok {
		switch rule {
		case ProductionCondition:
			condition, err := lowerCond(text, node)
			if err != nil {
				return err
			}
			dst.add(condition)
			return nil
		case ProductionDefine:
			fn, err := lowerDefine(text, node)
			if err != nil {
				return err
			}
			dst.add(fn)
			return nil
		default:
			return syn(lineAt(text, node.Start()), "syntax")
		}
	}
	term, ok := node.Term()
	if !ok {
		return syn(lineAt(text, node.Start()), "syntax")
	}
	raw := node.Text()
	line := cookedLine(raw)
	n := lineAt(text, node.Start())
	switch term {
	case TermRecipe:
		attachRecipe(dst, recipeText(raw))
		return nil
	case TermAssign:
		statement, err := lowerAssign(line, n)
		if err != nil {
			return err
		}
		dst.add(statement)
		return nil
	case TermRule:
		statement, err := lowerRule(line, n)
		if err != nil {
			return err
		}
		dst.add(statement)
		return nil
	case TermInclude:
		statement, err := lowerInclude(line, n)
		if err != nil {
			return err
		}
		dst.add(statement)
		return nil
	case TermVPath:
		dst.add(lowerVPath(line))
		return nil
	case TermExport:
		dst.add(lowerExport(line))
		return nil
	case TermUndefine:
		statement, err := lowerUndefine(line, n)
		if err != nil {
			return err
		}
		dst.add(statement)
		return nil
	case TermUnknown:
		return syn(n, "unknown statement %s", short(line))
	default:
		return syn(n, "syntax")
	}
}

func attachRecipe(dst adder, text string) {
	if rule, ok := dst.last().(*Rule); ok {
		rule.Recipe = append(rule.Recipe, text)
		return
	}
	dst.add(&Rule{Recipe: []string{text}})
}

func lowerAssign(line string, n int) (Stmt, error) {
	if a, ok := assignmentKind(line); ok && a.kind == assignVar {
		if a.def.name == "" {
			return nil, syn(n, "empty variable name")
		}
		return &Variable{
			Name:     a.def.name,
			Op:       a.def.op,
			Value:    a.def.value,
			Override: a.override,
			Export:   a.export,
			Private:  a.private,
		}, nil
	}
	return lowerRule(line, n)
}

func lowerRule(line string, n int) (Stmt, error) {
	sep, ok := findRuleSep(line)
	if !ok {
		return nil, syn(n, "unknown statement %s", short(line))
	}
	targets := words(line[:sep.start])
	right := line[sep.end:]
	if !sep.double && !sep.group {
		if a, ok := assignmentKind(trimLeft(right)); ok && a.kind == assignVar {
			if len(targets) == 0 {
				return nil, syn(n, "empty target")
			}
			if a.def.name == "" {
				return nil, syn(n, "empty variable name")
			}
			return &Variable{
				Name:     a.def.name,
				Op:       a.def.op,
				Value:    a.def.value,
				Targets:  targets,
				Override: a.override,
				Export:   a.export,
				Private:  a.private,
			}, nil
		}
	}
	if len(targets) == 0 {
		return nil, syn(n, "missing separator")
	}
	rule := &Rule{Targets: targets, DoubleColon: sep.double, Grouped: sep.group}
	if pattern, after, ok := splitStatic(right); ok {
		rule.Pattern = trimSpace(pattern)
		right = after
	}
	if head, recipe, ok := splitSemi(right); ok {
		right = head
		rule.Recipe = []string{recipe}
	}
	if head, order, ok := splitBar(right); ok {
		rule.Prerequisites = words(head)
		rule.OrderOnly = words(order)
	} else {
		rule.Prerequisites = words(right)
	}
	return rule, nil
}

func lowerInclude(line string, n int) (Stmt, error) {
	word, rest := cutWord(line)
	paths := words(rest)
	if len(paths) == 0 {
		return nil, syn(n, "empty include")
	}
	return &Include{Paths: paths, Optional: word != "include"}, nil
}

func lowerVPath(line string) *VPath {
	_, rest := cutWord(line)
	rest = trimLeft(rest)
	if rest == "" {
		return &VPath{ClearAll: true}
	}
	pattern, dirs := cutWord(rest)
	return &VPath{Pattern: pattern, Dirs: trimSpace(dirs)}
}

func lowerExport(line string) *Export {
	word, rest := cutWord(line)
	names := words(rest)
	ex := &Export{Drop: word == "unexport"}
	if len(names) == 0 {
		ex.All = true
	} else {
		ex.Names = names
	}
	return ex
}

func lowerUndefine(line string, n int) (Stmt, error) {
	rest := ""
	if a, ok := assignmentKind(line); ok && a.kind == assignUndefine {
		rest = a.rest
	} else {
		_, rest = cutWord(line)
	}
	names := words(rest)
	if len(names) == 0 {
		return nil, syn(n, "empty variable name")
	}
	return &Undefine{Names: names}, nil
}

func lowerCond(text string, node xparse.Syn[Term, Production]) (*Conditional, error) {
	condition := &Conditional{}
	var cur *Branch
	elseSeen := false
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if rule, ok := child.Rule(); ok && rule == ProductionElse {
			if elseSeen {
				return nil, syn(elseLine(text, child), "only one else per conditional")
			}
			br, plain, err := lowerElse(text, child)
			if err != nil {
				return nil, err
			}
			if plain {
				elseSeen = true
			}
			condition.Branches = append(condition.Branches, br)
			cur = br
			if err := appendElse(text, child, br); err != nil {
				return nil, err
			}
			continue
		}
		if term, ok := child.Term(); ok {
			switch term {
			case TermIf:
				br, err := lowerIf(text, child)
				if err != nil {
					return nil, err
				}
				condition.Branches = append(condition.Branches, br)
				cur = br
				continue
			case TermEndif:
				if err := checkEndif(text, child); err != nil {
					return nil, err
				}
				continue
			}
		}
		if cur == nil {
			return nil, syn(lineAt(text, child.Start()), "syntax")
		}
		if err := appendStmt(text, child, cur); err != nil {
			return nil, err
		}
	}
	if len(condition.Branches) == 0 {
		return nil, syn(lineAt(text, node.Start()), "invalid conditional")
	}
	return condition, nil
}

func appendElse(text string, node xparse.Syn[Term, Production], br *Branch) error {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermElse {
			continue
		}
		if err := appendStmt(text, child, br); err != nil {
			return err
		}
	}
	return nil
}

func lowerIf(text string, tok xparse.Syn[Term, Production]) (*Branch, error) {
	line := cookedLine(tok.Text())
	word, rest := cutWord(line)
	return testBranch(word, rest, lineAt(text, tok.Start()))
}

func lowerElse(text string, node xparse.Syn[Term, Production]) (*Branch, bool, error) {
	tok := elseTok(node)
	line := cookedLine(tok.Text())
	n := lineAt(text, tok.Start())
	_, rest := cutWord(line)
	rest = trimLeft(rest)
	if rest == "" {
		return &Branch{}, true, nil
	}
	word, erest := cutWord(rest)
	switch word {
	case "ifeq", "ifneq", "ifdef", "ifndef":
		br, err := testBranch(word, erest, n)
		return br, false, err
	default:
		return nil, false, syn(n, "extraneous text after else")
	}
}

func elseTok(node xparse.Syn[Term, Production]) xparse.Syn[Term, Production] {
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		if term, ok := child.Term(); ok && term == TermElse {
			return child
		}
	}
	return node
}

func elseLine(text string, node xparse.Syn[Term, Production]) int {
	return lineAt(text, elseTok(node).Start())
}

func checkEndif(text string, tok xparse.Syn[Term, Production]) error {
	_, rest := cutWord(cookedLine(tok.Text()))
	if trimSpace(rest) != "" {
		return syn(lineAt(text, tok.Start()), "extraneous text after endif")
	}
	return nil
}

func lowerDefine(text string, node xparse.Syn[Term, Production]) (Stmt, error) {
	var (
		header xparse.Syn[Term, Production]
		endef  xparse.Syn[Term, Production]
		body   []string
	)
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		term, ok := child.Term()
		if !ok {
			return nil, syn(lineAt(text, child.Start()), "syntax")
		}
		switch term {
		case TermDefine:
			header = child
		case TermBody:
			body = append(body, bodyLine(child.Text()))
		case TermEndef:
			endef = child
		default:
			return nil, syn(lineAt(text, child.Start()), "syntax")
		}
	}
	if !header.Valid() {
		return nil, syn(lineAt(text, node.Start()), "missing endef")
	}
	fn, err := defineHeader(cookedLine(header.Text()), lineAt(text, header.Start()))
	if err != nil {
		return nil, err
	}
	if !endef.Valid() {
		return nil, syn(lineAt(text, header.Start()), "missing endef")
	}
	if err := checkEndef(text, endef); err != nil {
		return nil, err
	}
	fn.Body = strings.Join(body, "\n")
	return fn, nil
}

func defineHeader(line string, n int) (*Function, error) {
	a, ok := assignmentKind(line)
	if !ok || a.kind != assignDefine {
		return nil, syn(n, "syntax")
	}
	fn := &Function{Override: a.override, Export: a.export, Private: a.private}
	if def, ok := parseVariableDefinition(a.rest); ok {
		if trimSpace(def.value) != "" {
			return nil, syn(n, "extraneous text after define")
		}
		fn.Name = def.name
		fn.Op = def.op
	} else {
		fn.Name = trimSpace(a.rest)
	}
	if fn.Name == "" {
		return nil, syn(n, "empty variable name")
	}
	return fn, nil
}

func checkEndef(text string, tok xparse.Syn[Term, Production]) error {
	line := collapse(tok.Text())
	n := lineAt(text, tok.Start())
	if strings.HasPrefix(line, "\t") {
		return syn(n, "extraneous text after endef")
	}
	_, rest := cutWord(trimLeft(line))
	if trimSpace(removeComments(rest)) != "" {
		return syn(n, "extraneous text after endef")
	}
	return nil
}

func lineAt(text string, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(text) {
		off = len(text)
	}
	return strings.Count(text[:off], "\n") + 1
}

type assignKind int

const (
	assignVar assignKind = iota
	assignDefine
	assignUndefine
)

type assignment struct {
	kind     assignKind
	rest     string
	def      varDef
	override bool
	export   bool
	private  bool
}

func assignmentKind(line string) (assignment, bool) {
	var a assignment
	p := line
	for {
		if def, ok := parseVariableDefinition(p); ok {
			a.kind = assignVar
			a.def = def
			return a, true
		}
		word, rest := cutWord(p)
		switch word {
		case "override":
			a.override = true
		case "export":
			a.export = true
		case "private":
			a.private = true
		case "define":
			a.kind = assignDefine
			a.rest = trimLeft(rest)
			return a, true
		case "undefine":
			a.kind = assignUndefine
			a.rest = trimLeft(rest)
			return a, true
		default:
			return assignment{}, false
		}
		p = trimLeft(rest)
		if p == "" {
			return assignment{}, false
		}
	}
}

func testBranch(word, rest string, n int) (*Branch, error) {
	br := &Branch{HasTest: true}
	switch word {
	case "ifdef":
		br.Kind = KindIfDef
		return oneName(br, rest, n)
	case "ifndef":
		br.Kind = KindIfNdef
		return oneName(br, rest, n)
	case "ifeq":
		br.Kind = KindIfEq
	case "ifneq":
		br.Kind = KindIfNeq
	default:
		return nil, syn(n, "invalid conditional")
	}
	left, right, err := parseIfArgs(rest)
	if err != nil {
		return nil, syn(n, "invalid syntax in conditional")
	}
	br.Left = left
	br.Right = right
	return br, nil
}

func oneName(br *Branch, rest string, n int) (*Branch, error) {
	name, extra := cutWord(rest)
	if name == "" || trimSpace(extra) != "" {
		return nil, syn(n, "invalid syntax in conditional")
	}
	br.Left = name
	return br, nil
}

func parseIfArgs(rest string) (string, string, error) {
	rest = trimLeft(rest)
	if rest == "" {
		return "", "", errSyntax
	}
	switch rest[0] {
	case '(':
		return parseParenArgs(rest)
	case '"', '\'':
		return parseQuotedArgs(rest)
	default:
		return "", "", errSyntax
	}
}

func parseParenArgs(rest string) (string, string, error) {
	depth := 0
	comma := -1
	close := -1
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				close = i
				i = len(rest)
			}
		case ',':
			if depth == 1 && comma < 0 {
				comma = i
			}
		}
	}
	if close < 0 || comma < 0 || trimSpace(rest[close+1:]) != "" {
		return "", "", errSyntax
	}
	left := trimRight(rest[1:comma])
	right := trimSpace(rest[comma+1 : close])
	return left, right, nil
}

func parseQuotedArgs(rest string) (string, string, error) {
	q := rest[0]
	end := strings.IndexByte(rest[1:], q)
	if end < 0 {
		return "", "", errSyntax
	}
	end++
	left := rest[1:end]
	rest = trimLeft(rest[end+1:])
	if rest == "" {
		return "", "", errSyntax
	}
	q = rest[0]
	if q != '"' && q != '\'' {
		return "", "", errSyntax
	}
	end = strings.IndexByte(rest[1:], q)
	if end < 0 {
		return "", "", errSyntax
	}
	end++
	right := rest[1:end]
	if trimSpace(rest[end+1:]) != "" {
		return "", "", errSyntax
	}
	return left, right, nil
}

func syn(n int, format string, args ...any) error {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return fmt.Errorf("%w: line %d: %s", errSyntax, n, msg)
}
