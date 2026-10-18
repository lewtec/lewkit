package ninja

import (
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Production]
	langErr  error
)

//go:generate go run github.com/lewtec/lewkit/cmd/lewkit generate parse

// Language loads the ninja grammar. The scanner is the ninja phase lexer.
func Language() (*xparse.Model[Term, Production], error) {
	langOnce.Do(func() {
		prepared, err := declare()
		if err != nil {
			langErr = err
			return
		}
		lang, langErr = prepared.Load(ninjaTable)
	})
	return lang, langErr
}

func declare() (*xparse.Prepared[Term, Production], error) {
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
	return syntax.Prepare("ninja", syntax.Compose(
		file, variable, ruleDecl, poolDecl, edge, defaultDecl, includeDecl, subDecl,
		binding, outputs, inputs,
		syntax.Scan(scanSrc),
		syntax.Start(ProductionFile),
	))
}

// parse reads a ninja manifest the way ninja 1.12's lexer does.
// Comments and blank lines are dropped. Include paths are not loaded.
