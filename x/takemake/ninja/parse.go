package ninja

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
		lang, langErr = define()
	})
	return lang, langErr
}

func define() (*xparse.Model[Term, Production], error) {
	var syntax xparse.Syntax[Term, Production]
	identifier := syntax.Word(TermIdentifier, wordRune, wordRune)
	build := syntax.Key(TermBuild, TermIdentifier)
	pool := syntax.Key(TermPool, TermIdentifier)
	rule := syntax.Key(TermRule, TermIdentifier)
	defaultKeyword := syntax.Key(TermDefault, TermIdentifier)
	includeKeyword := syntax.Key(TermInclude, TermIdentifier)
	subninjaKeyword := syntax.Key(TermSubninja, TermIdentifier)
	equal := syntax.Term(TermEqual, nil)
	colon := syntax.Term(TermColon, nil)
	pipe := syntax.Term(TermPipe, nil)
	doublePipe := syntax.Term(TermDoublePipe, nil)
	pipeAt := syntax.Term(TermPipeAtSign, nil)
	indent := syntax.Term(TermIndent, nil)
	newline := syntax.Term(TermNewline, nil)
	value := syntax.Term(TermValue, nil)
	path := syntax.Term(TermPath, nil)

	paths := syntax.Repeat(path)
	binding := syntax.Eq(ProductionBinding, syntax.Seq(indent, identifier, equal, syntax.Opt(value), newline))
	variable := syntax.Eq(ProductionVariable, syntax.Seq(identifier, equal, syntax.Opt(value), newline))
	ruleDecl := syntax.Eq(ProductionRule, syntax.Seq(rule, identifier, newline, syntax.Repeat(syntax.Ref(ProductionBinding))))
	poolDecl := syntax.Eq(ProductionPool, syntax.Seq(pool, identifier, newline, syntax.Repeat(syntax.Ref(ProductionBinding))))
	outputs := syntax.Eq(ProductionOutputs, syntax.Alt(
		syntax.Seq(syntax.Repeat1(path), syntax.Opt(syntax.Seq(pipe, paths))),
		syntax.Seq(pipe, syntax.Repeat1(path)),
	))
	inputs := syntax.Eq(ProductionInputs, syntax.Seq(
		paths,
		syntax.Opt(syntax.Seq(pipe, paths)),
		syntax.Opt(syntax.Seq(doublePipe, paths)),
		syntax.Opt(syntax.Seq(pipeAt, paths)),
	))
	edge := syntax.Eq(ProductionEdge, syntax.Seq(
		build, syntax.Ref(ProductionOutputs), colon, identifier, syntax.Ref(ProductionInputs), newline, syntax.Repeat(syntax.Ref(ProductionBinding)),
	))
	defaultDecl := syntax.Eq(ProductionDefault, syntax.Seq(defaultKeyword, syntax.Repeat1(path), newline))
	includeDecl := syntax.Eq(ProductionInclude, syntax.Seq(includeKeyword, path, newline))
	subDecl := syntax.Eq(ProductionSubninja, syntax.Seq(subninjaKeyword, path, newline))
	file := syntax.Eq(ProductionFile, syntax.Repeat(syntax.Alt(
		newline,
		syntax.Ref(ProductionVariable),
		syntax.Ref(ProductionRule),
		syntax.Ref(ProductionPool),
		syntax.Ref(ProductionEdge),
		syntax.Ref(ProductionDefault),
		syntax.Ref(ProductionInclude),
		syntax.Ref(ProductionSubninja),
	)))
	return syntax.Compile("ninja", syntax.Compose(
		file, variable, ruleDecl, poolDecl, edge, defaultDecl, includeDecl, subDecl,
		binding, outputs, inputs,
		syntax.Scan(scanSrc),
		syntax.Start(ProductionFile),
	))
}

// parse reads a ninja manifest the way ninja 1.12's lexer does.
// Comments and blank lines are dropped. Include paths are not loaded.
func parse(s string) (*File, error) {
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(s))
	if err != nil {
		return nil, parseErr(s, err)
	}
	return lower(s, root)
}

func parseErr(s string, err error) error {
	var pe *xparse.Error
	if !errors.As(err, &pe) {
		return err
	}
	if lexErr := firstLexErr(s); lexErr != nil {
		return lexErr
	}
	return fmt.Errorf("%w: line %d: syntax", errSyntax, pe.Line)
}

func lower(src string, root xparse.Syn[Term, Production]) (*File, error) {
	rule, ok := root.Rule()
	if !ok || rule != ProductionFile {
		return nil, fmt.Errorf("%w: line 1: syntax", errSyntax)
	}
	file := New()
	for i := 0; i < root.ChildCount(); i++ {
		child := root.Child(i)
		term, isTerm := child.Term()
		if isTerm {
			if term == TermNewline {
				continue
			}
			return nil, badNode(src, child)
		}
		stmt, err := lowerStmt(src, child)
		if err != nil {
			return nil, err
		}
		file.Stmts = append(file.Stmts, stmt)
	}
	return file, nil
}

func lowerStmt(src string, n xparse.Syn[Term, Production]) (Stmt, error) {
	rule, ok := n.Rule()
	if !ok {
		return nil, badNode(src, n)
	}
	switch rule {
	case ProductionVariable:
		name, value, err := lowerBind(src, n)
		if err != nil {
			return nil, err
		}
		return &Variable{Name: name, Value: value}, nil
	case ProductionRule:
		r := &Rule{}
		if err := lowerNamed(src, n, &r.Name, &r.Bindings); err != nil {
			return nil, err
		}
		return r, nil
	case ProductionPool:
		p := &Pool{}
		if err := lowerNamed(src, n, &p.Name, &p.Bindings); err != nil {
			return nil, err
		}
		return p, nil
	case ProductionEdge:
		return lowerEdge(src, n)
	case ProductionDefault:
		return lowerDefault(src, n)
	case ProductionInclude:
		return lowerInclude(src, n, false)
	case ProductionSubninja:
		return lowerInclude(src, n, true)
	default:
		return nil, badNode(src, n)
	}
}

func lowerNamed(src string, n xparse.Syn[Term, Production], name *string, binds *[]Binding) error {
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if rule, ok := child.Rule(); ok && rule == ProductionBinding {
			b, err := lowerBinding(src, child)
			if err != nil {
				return err
			}
			*binds = append(*binds, b)
			continue
		}
		term, ok := child.Term()
		if !ok {
			return badNode(src, child)
		}
		if term == TermIdentifier && *name == "" {
			*name = child.Text()
		}
	}
	if *name == "" {
		return badNode(src, n)
	}
	return nil
}

func lowerEdge(src string, n xparse.Syn[Term, Production]) (*Edge, error) {
	e := &Edge{}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		if rule, ok := child.Rule(); ok {
			switch rule {
			case ProductionOutputs:
				outs, impl, err := splitOuts(src, child)
				if err != nil {
					return nil, err
				}
				e.Outputs, e.ImplicitOutputs = outs, impl
			case ProductionInputs:
				ins, impl, order, valid, err := splitIns(src, child)
				if err != nil {
					return nil, err
				}
				e.Inputs, e.ImplicitDeps, e.OrderOnlyDeps, e.Validations = ins, impl, order, valid
			case ProductionBinding:
				b, err := lowerBinding(src, child)
				if err != nil {
					return nil, err
				}
				e.Bindings = append(e.Bindings, b)
			default:
				return nil, badNode(src, child)
			}
			continue
		}
		term, ok := child.Term()
		if !ok {
			return nil, badNode(src, child)
		}
		if term == TermIdentifier {
			e.Rule = child.Text()
		}
	}
	if e.Rule == "" {
		return nil, badNode(src, n)
	}
	return e, nil
}

func lowerDefault(src string, n xparse.Syn[Term, Production]) (*Default, error) {
	d := &Default{}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		term, ok := child.Term()
		if !ok || term != TermPath {
			continue
		}
		ps, err := piecesAt(src, child, true)
		if err != nil {
			return nil, err
		}
		d.Targets = append(d.Targets, ps)
	}
	if len(d.Targets) == 0 {
		return nil, badNode(src, n)
	}
	return d, nil
}

func lowerInclude(src string, n xparse.Syn[Term, Production], subninja bool) (*Include, error) {
	include := &Include{Subninja: subninja}
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		term, ok := child.Term()
		if !ok || term != TermPath {
			continue
		}
		ps, err := piecesAt(src, child, true)
		if err != nil {
			return nil, err
		}
		include.Path = ps
	}
	if len(include.Path) == 0 {
		return nil, badNode(src, n)
	}
	return include, nil
}

func lowerBinding(src string, n xparse.Syn[Term, Production]) (Binding, error) {
	name, value, err := lowerBind(src, n)
	if err != nil {
		return Binding{}, err
	}
	return Binding{Name: name, Value: value}, nil
}

func lowerBind(src string, n xparse.Syn[Term, Production]) (string, []Piece, error) {
	var name string
	var value []Piece
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		term, ok := child.Term()
		if !ok {
			return "", nil, badNode(src, child)
		}
		switch term {
		case TermIdentifier:
			if name == "" {
				name = child.Text()
			}
		case TermValue:
			ps, err := piecesAt(src, child, false)
			if err != nil {
				return "", nil, err
			}
			value = ps
		}
	}
	if name == "" {
		return "", nil, badNode(src, n)
	}
	return name, value, nil
}

func splitOuts(src string, n xparse.Syn[Term, Production]) ([][]Piece, [][]Piece, error) {
	var explicit, implicit [][]Piece
	cur := &explicit
	saw := false
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		term, ok := child.Term()
		if !ok {
			return nil, nil, badNode(src, child)
		}
		switch term {
		case TermPath:
			ps, err := piecesAt(src, child, true)
			if err != nil {
				return nil, nil, err
			}
			*cur = append(*cur, ps)
		case TermPipe:
			cur = &implicit
			saw = true
		default:
			return nil, nil, badNode(src, child)
		}
	}
	if !saw {
		return explicit, nil, nil
	}
	return explicit, implicit, nil
}

func splitIns(src string, n xparse.Syn[Term, Production]) ([][]Piece, [][]Piece, [][]Piece, [][]Piece, error) {
	var buckets [4][][]Piece
	bucket := 0
	for i := 0; i < n.ChildCount(); i++ {
		child := n.Child(i)
		term, ok := child.Term()
		if !ok {
			return nil, nil, nil, nil, badNode(src, child)
		}
		switch term {
		case TermPath:
			ps, err := piecesAt(src, child, true)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			buckets[bucket] = append(buckets[bucket], ps)
		case TermPipe:
			bucket = 1
		case TermDoublePipe:
			bucket = 2
		case TermPipeAtSign:
			bucket = 3
		default:
			return nil, nil, nil, nil, badNode(src, child)
		}
	}
	return buckets[0], buckets[1], buckets[2], buckets[3], nil
}

func piecesAt(src string, n xparse.Syn[Term, Production], path bool) ([]Piece, error) {
	ps, err := evalText(n.Text(), path)
	if err == nil {
		return ps, nil
	}
	msg := "syntax"
	if text := err.Error(); text != "" {
		if i := strings.LastIndex(text, ": "); i >= 0 {
			msg = text[i+2:]
		}
	}
	return nil, fmt.Errorf("%w: line %d: %s", errSyntax, lineAt(src, n.Start()), msg)
}

func badNode(src string, n xparse.Syn[Term, Production]) error {
	return fmt.Errorf("%w: line %d: syntax", errSyntax, lineAt(src, n.Start()))
}

func lineAt(src string, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	return strings.Count(src[:off], "\n") + 1
}
