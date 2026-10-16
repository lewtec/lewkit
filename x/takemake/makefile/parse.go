package makefile

import (
	"fmt"
	"strings"
)

type adder interface {
	add(Stmt)
	last() Stmt
	top() bool
}

type frame struct {
	prev     adder
	cond     *Conditional
	elseSeen bool
}

type parser struct {
	lines []string
	i     int
	cur   adder
	stack []frame
}

func parse(text string) (*File, error) {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	f := New()
	p := &parser{lines: strings.Split(text, "\n"), cur: f}
	if err := p.parse(); err != nil {
		return nil, err
	}
	return f, nil
}

func (p *parser) parse() error {
	for p.i < len(p.lines) {
		if strings.HasPrefix(p.lines[p.i], "\t") && p.tabIsRecipe() {
			text := p.lines[p.i][1:]
			p.i++
			p.attachRecipe(text)
			continue
		}
		n := p.i + 1
		raw, err := p.readLogical()
		if err != nil {
			return err
		}
		line := trimLeft(removeComments(collapse(raw)))
		if err := p.statement(line, n); err != nil {
			return err
		}
	}
	if len(p.stack) > 0 {
		return syn(len(p.lines), "missing endif")
	}
	return nil
}

func (p *parser) tabIsRecipe() bool {
	if r, ok := p.cur.last().(*Rule); ok && len(r.Targets) > 0 {
		return true
	}
	return !p.cur.top()
}

func (p *parser) attachRecipe(text string) {
	if r, ok := p.cur.last().(*Rule); ok {
		r.Recipe = append(r.Recipe, text)
		return
	}
	p.cur.add(&Rule{Recipe: []string{text}})
}

func (p *parser) readLogical() (string, error) {
	var b strings.Builder
	for {
		if p.i >= len(p.lines) {
			return "", syn(p.i, "unterminated continuation")
		}
		line := p.lines[p.i]
		p.i++
		b.WriteString(line)
		if trailingBackslashes(line)%2 == 0 {
			return b.String(), nil
		}
		if p.i >= len(p.lines) {
			return "", syn(p.i, "unterminated continuation")
		}
		b.WriteByte('\n')
	}
}

func (p *parser) statement(line string, n int) error {
	if line == "" {
		return nil
	}
	if a, ok := assignmentKind(line); ok {
		return p.assign(a, n)
	}
	word, rest := cutWord(line)
	switch word {
	case "ifeq", "ifneq", "ifdef", "ifndef", "else", "endif":
		return p.conditional(word, rest, n)
	case "include", "-include", "sinclude":
		return p.include(word, rest, n)
	case "vpath":
		return p.vpath(rest, n)
	case "export", "unexport":
		return p.export(word == "unexport", rest)
	case "undefine":
		return p.undefine(rest, n)
	}
	if _, ok := findRuleSep(line); ok {
		return p.rule(line, n)
	}
	return syn(n, "unknown statement %s", short(line))
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

func (p *parser) assign(a assignment, n int) error {
	switch a.kind {
	case assignDefine:
		return p.define(a, n)
	case assignUndefine:
		return p.undefine(a.rest, n)
	default:
		if a.def.name == "" {
			return syn(n, "empty variable name")
		}
		p.cur.add(&Variable{
			Name:     a.def.name,
			Op:       a.def.op,
			Value:    a.def.value,
			Override: a.override,
			Export:   a.export,
			Private:  a.private,
		})
		return nil
	}
}

func (p *parser) define(a assignment, n int) error {
	fn := &Function{Override: a.override, Export: a.export, Private: a.private}
	if def, ok := parseVariableDefinition(a.rest); ok {
		if trimSpace(def.value) != "" {
			return syn(n, "extraneous text after define")
		}
		fn.Name = def.name
		fn.Op = def.op
	} else {
		fn.Name = trimSpace(a.rest)
	}
	if fn.Name == "" {
		return syn(n, "empty variable name")
	}
	body, err := p.readBody(n)
	if err != nil {
		return err
	}
	fn.Body = body
	p.cur.add(fn)
	return nil
}

func (p *parser) readBody(n int) (string, error) {
	var lines []string
	depth := 1
	for {
		if p.i >= len(p.lines) {
			return "", syn(n, "missing endef")
		}
		lineNo := p.i + 1
		raw, err := p.readLogical()
		if err != nil {
			return "", err
		}
		line := collapse(raw)
		if !strings.HasPrefix(line, "\t") {
			word, rest := cutWord(trimLeft(line))
			switch word {
			case "define":
				depth++
			case "endef":
				if trimSpace(removeComments(rest)) != "" {
					return "", syn(lineNo, "extraneous text after endef")
				}
				depth--
				if depth == 0 {
					return strings.Join(lines, "\n"), nil
				}
			}
		}
		lines = append(lines, line)
	}
}

func (p *parser) undefine(rest string, n int) error {
	names := words(rest)
	if len(names) == 0 {
		return syn(n, "empty variable name")
	}
	p.cur.add(&Undefine{Names: names})
	return nil
}

func (p *parser) conditional(word, rest string, n int) error {
	switch word {
	case "endif":
		if trimSpace(rest) != "" {
			return syn(n, "extraneous text after endif")
		}
		if len(p.stack) == 0 {
			return syn(n, "extraneous endif")
		}
		fr := p.stack[len(p.stack)-1]
		p.stack = p.stack[:len(p.stack)-1]
		p.cur = fr.prev
		return nil
	case "else":
		if len(p.stack) == 0 {
			return syn(n, "extraneous else")
		}
		fr := &p.stack[len(p.stack)-1]
		if fr.elseSeen {
			return syn(n, "only one else per conditional")
		}
		rest = trimLeft(rest)
		if rest == "" {
			fr.elseSeen = true
			br := &Branch{}
			fr.cond.Branches = append(fr.cond.Branches, br)
			p.cur = br
			return nil
		}
		ew, erest := cutWord(rest)
		switch ew {
		case "ifeq", "ifneq", "ifdef", "ifndef":
			br, err := testBranch(ew, erest, n)
			if err != nil {
				return err
			}
			fr.cond.Branches = append(fr.cond.Branches, br)
			p.cur = br
			return nil
		default:
			return syn(n, "extraneous text after else")
		}
	default:
		br, err := testBranch(word, rest, n)
		if err != nil {
			return err
		}
		cond := &Conditional{Branches: []*Branch{br}}
		p.cur.add(cond)
		p.stack = append(p.stack, frame{prev: p.cur, cond: cond})
		p.cur = br
		return nil
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

func (p *parser) include(word, rest string, n int) error {
	paths := words(rest)
	if len(paths) == 0 {
		return syn(n, "empty include")
	}
	p.cur.add(&Include{Paths: paths, Optional: word != "include"})
	return nil
}

func (p *parser) vpath(rest string, _ int) error {
	rest = trimLeft(rest)
	if rest == "" {
		p.cur.add(&VPath{ClearAll: true})
		return nil
	}
	pattern, dirs := cutWord(rest)
	p.cur.add(&VPath{Pattern: pattern, Dirs: trimSpace(dirs)})
	return nil
}

func (p *parser) export(drop bool, rest string) error {
	names := words(rest)
	ex := &Export{Drop: drop}
	if len(names) == 0 {
		ex.All = true
	} else {
		ex.Names = names
	}
	p.cur.add(ex)
	return nil
}

func (p *parser) rule(line string, n int) error {
	sep, ok := findRuleSep(line)
	if !ok {
		return syn(n, "unknown statement %s", short(line))
	}
	targets := words(line[:sep.start])
	right := line[sep.end:]
	if !sep.double && !sep.group {
		if a, ok := assignmentKind(trimLeft(right)); ok && a.kind == assignVar {
			if len(targets) == 0 {
				return syn(n, "empty target")
			}
			if a.def.name == "" {
				return syn(n, "empty variable name")
			}
			p.cur.add(&Variable{
				Name:     a.def.name,
				Op:       a.def.op,
				Value:    a.def.value,
				Targets:  targets,
				Override: a.override,
				Export:   a.export,
				Private:  a.private,
			})
			return nil
		}
	}
	if len(targets) == 0 {
		return syn(n, "missing separator")
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
	p.cur.add(rule)
	return nil
}

func syn(n int, format string, args ...any) error {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	return fmt.Errorf("%w: line %d: %s", errSyntax, n, msg)
}
