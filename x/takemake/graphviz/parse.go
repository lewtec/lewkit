package graphviz

import (
	"errors"
	"fmt"
	"sync"

	xparse "github.com/lewtec/lewkit/x/parse"
)

var (
	langOnce sync.Once
	lang     *xparse.Model[Term, Rule]
	langErr  error
)

func language() (*xparse.Model[Term, Rule], error) {
	langOnce.Do(func() {
		lang, langErr = define()
	})
	return lang, langErr
}

func define() (*xparse.Model[Term, Rule], error) {
	var syntax xparse.Syntax[Term, Rule]

	// Key beats an equal-length identifier. The identifier continuation keeps a keyword
	// from matching inside a longer word. Num never matches those bytes.
	identifier := syntax.Word(TermIdentifier, identifierStart, identifierContinue)
	number := syntax.Term(TermNumber, matchNumber)
	stringToken := syntax.Term(TermString, matchString)
	htmlLabel := syntax.Term(TermHTMLLabel, matchHTML)
	strict := syntax.Key(TermStrict, TermIdentifier)
	graph := syntax.Key(TermGraph, TermIdentifier)
	digraph := syntax.Key(TermDigraph, TermIdentifier)
	subgraph := syntax.Key(TermSubgraph, TermIdentifier)
	node := syntax.Key(TermNode, TermIdentifier)
	edge := syntax.Key(TermEdge, TermIdentifier)

	leftBrace := syntax.Lit("{")
	rightBrace := syntax.Lit("}")
	leftBracket := syntax.Lit("[")
	rightBracket := syntax.Lit("]")
	semicolon := syntax.Lit(";")
	comma := syntax.Lit(",")
	equal := syntax.Lit("=")
	colon := syntax.Lit(":")
	plus := syntax.Lit("+")
	edgeop := syntax.Alt(syntax.Lit("->"), syntax.Lit("--"))
	// A separator may be absent or trailing. The comma and the following
	// pair stay in one item so a trailing separator does not also parse
	// as the start of another pair.
	sep := syntax.Alt(comma, semicolon)

	atom := syntax.Eq(RuleAtom, syntax.Alt(identifier, number, syntax.Ref(RuleStrings)))
	strings := syntax.Eq(RuleStrings, syntax.Seq(
		syntax.Alt(stringToken, htmlLabel),
		syntax.Repeat(syntax.Seq(plus, syntax.Alt(stringToken, htmlLabel))),
	))
	pair := syntax.Eq(RulePair, syntax.Seq(syntax.Ref(RuleAtom), equal, syntax.Ref(RuleAtom)))
	more := syntax.Eq(RuleMore, syntax.Alt(
		syntax.Seq(sep, syntax.Ref(RulePair), syntax.Ref(RuleMore)),
		syntax.Seq(syntax.Ref(RulePair), syntax.Ref(RuleMore)),
		sep,
		syntax.Seq(),
	))
	attributeList := syntax.Eq(RuleAttributeList, syntax.Seq(leftBracket, syntax.Ref(RuleMore), rightBracket))
	port := syntax.Eq(RulePort, syntax.Seq(
		colon,
		syntax.Ref(RuleAtom),
		syntax.Opt(syntax.Seq(colon, syntax.Ref(RuleAtom))),
	))
	nodeIdentifier := syntax.Eq(RuleNodeIdentifier, syntax.Seq(syntax.Ref(RuleAtom), syntax.Opt(syntax.Ref(RulePort))))
	identifiers := syntax.Eq(RuleIdentifiers, syntax.Sep1(syntax.Ref(RuleNodeIdentifier), comma))
	block := syntax.Eq(RuleBlock, syntax.Alt(
		syntax.Seq(subgraph, syntax.Opt(syntax.Ref(RuleAtom)), leftBrace, syntax.Repeat(syntax.Ref(RuleStatement)), rightBrace),
		syntax.Seq(leftBrace, syntax.Repeat(syntax.Ref(RuleStatement)), rightBrace),
	))
	end := syntax.Eq(RuleEnd, syntax.Alt(syntax.Ref(RuleIdentifiers), syntax.Ref(RuleBlock)))
	edgeTail := syntax.Eq(RuleEdgeTail, syntax.Alt(
		syntax.Seq(edgeop, syntax.Ref(RuleEnd), syntax.Ref(RuleEdgeTail)),
		syntax.Repeat(syntax.Ref(RuleAttributeList)),
	))
	list := syntax.Eq(RuleList, syntax.Alt(
		syntax.Seq(comma, syntax.Ref(RuleNodeIdentifier), syntax.Ref(RuleList)),
		syntax.Seq(edgeop, syntax.Ref(RuleEnd), syntax.Ref(RuleEdgeTail)),
		syntax.Repeat(syntax.Ref(RuleAttributeList)),
	))
	tail := syntax.Eq(RuleTail, syntax.Alt(
		syntax.Seq(equal, syntax.Ref(RuleAtom)),
		syntax.Seq(syntax.Opt(syntax.Ref(RulePort)), syntax.Ref(RuleList)),
	))
	lead := syntax.Eq(RuleLead, syntax.Seq(syntax.Ref(RuleAtom), syntax.Ref(RuleTail)))
	statementTail := syntax.Eq(RuleStatementTail, syntax.Opt(syntax.Seq(edgeop, syntax.Ref(RuleEnd), syntax.Ref(RuleEdgeTail))))
	subgraphStatement := syntax.Eq(RuleSubgraphStatement, syntax.Seq(syntax.Ref(RuleBlock), syntax.Ref(RuleStatementTail)))
	attributeStatement := syntax.Eq(RuleAttributeStatement, syntax.Seq(
		syntax.Alt(graph, node, edge),
		syntax.Opt(syntax.Seq(syntax.Ref(RuleAtom), equal)),
		syntax.Repeat1(syntax.Ref(RuleAttributeList)),
	))
	statement := syntax.Eq(RuleStatement, syntax.Seq(
		syntax.Alt(syntax.Ref(RuleAttributeStatement), syntax.Ref(RuleLead), syntax.Ref(RuleSubgraphStatement)),
		syntax.Opt(semicolon),
	))
	top := syntax.Eq(RuleTop, syntax.Seq(
		syntax.Opt(strict),
		syntax.Alt(graph, digraph),
		syntax.Opt(syntax.Ref(RuleAtom)),
		leftBrace,
		syntax.Repeat(syntax.Ref(RuleStatement)),
		rightBrace,
	))
	file := syntax.Eq(RuleFile, syntax.Repeat(syntax.Ref(RuleTop)))
	return syntax.Compile("graphviz", syntax.Compose(
		file, top, statement, attributeStatement, lead, tail, list, edgeTail, statementTail, subgraphStatement, block, end, identifiers, nodeIdentifier, port, attributeList, more, pair, atom, strings,
		syntax.Skip(syntax.Term(TermSkip, matchTrivia)),
		syntax.Start(RuleFile),
	))
}

func parseSource(src []byte) (*File, error) {
	src = stripBOM(src)
	model, err := language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse(src)
	if err != nil {
		return nil, syntaxParse(err)
	}
	return lowerFile(src, root)
}

func stripBOM(src []byte) []byte {
	if len(src) >= 3 && src[0] == 0xEF && src[1] == 0xBB && src[2] == 0xBF {
		return src[3:]
	}
	return src
}

func syntaxParse(err error) error {
	var pe *xparse.Error
	if errors.As(err, &pe) && pe.Line > 0 {
		return fmt.Errorf("%w: line %d", ErrSyntax, pe.Line)
	}
	return fmt.Errorf("%w", ErrSyntax)
}

func syntaxAt(src []byte, n xparse.Syn[Term, Rule]) error {
	line := 1
	if n.Valid() {
		line = lineAt(src, n.Start())
	}
	return fmt.Errorf("%w: line %d", ErrSyntax, line)
}

func lineAt(src []byte, off int) int {
	if off < 0 {
		off = 0
	}
	if off > len(src) {
		off = len(src)
	}
	n := 1
	for _, c := range src[:off] {
		if c == '\n' {
			n++
		}
	}
	return n
}

func identifierStart(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || r >= 0x80
}

func identifierContinue(r rune) bool {
	return identifierStart(r) || (r >= '0' && r <= '9')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func matchNumber(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	i := at
	switch src[i] {
	case '-':
		if !numAfterSign(src, i+1) {
			return 0
		}
		i++
	case '.':
		if i+1 >= len(src) || !isDigit(src[i+1]) {
			return 0
		}
		i++
	default:
		if !isDigit(src[i]) {
			return 0
		}
	}
	if i < len(src) && isDigit(src[i]) {
		for i < len(src) && isDigit(src[i]) {
			i++
		}
		if i < len(src) && src[i] == '.' {
			i++
			for i < len(src) && isDigit(src[i]) {
				i++
			}
		}
		return i - at
	}
	if i < len(src) && src[i] == '.' {
		i++
		if i >= len(src) || !isDigit(src[i]) {
			return 0
		}
		for i < len(src) && isDigit(src[i]) {
			i++
		}
		return i - at
	}
	return 0
}

func numAfterSign(src []byte, i int) bool {
	if i >= len(src) {
		return false
	}
	if isDigit(src[i]) {
		return true
	}
	return src[i] == '.' && i+1 < len(src) && isDigit(src[i+1])
}

func matchString(src []byte, at int) int {
	if at >= len(src) || src[at] != '"' {
		return 0
	}
	i := at + 1
	for i < len(src) {
		if src[i] == '"' {
			return i + 1 - at
		}
		if src[i] == '\\' && i+1 < len(src) {
			i += 2
			continue
		}
		i++
	}
	return 0
}

func matchHTML(src []byte, at int) int {
	if at >= len(src) || src[at] != '<' {
		return 0
	}
	i := at + 1
	nest := 1
	for i < len(src) {
		c := src[i]
		i++
		switch c {
		case '>':
			nest--
			if nest == 0 {
				return i - at
			}
		case '<':
			nest++
		}
	}
	return 0
}

func matchTrivia(src []byte, at int) int {
	if at >= len(src) {
		return 0
	}
	switch src[at] {
	case ' ', '\t', '\r', '\n':
		i := at + 1
		for i < len(src) {
			switch src[i] {
			case ' ', '\t', '\r', '\n':
				i++
			default:
				return i - at
			}
		}
		return i - at
	case '#':
		return runLine(src, at)
	case '/':
		if at+1 < len(src) && src[at+1] == '/' {
			return runLine(src, at)
		}
		if at+1 < len(src) && src[at+1] == '*' {
			return runBlock(src, at)
		}
	}
	return 0
}

func runLine(src []byte, at int) int {
	i := at
	for i < len(src) && src[i] != '\n' {
		i++
	}
	if i == at {
		return 0
	}
	return i - at
}

func runBlock(src []byte, at int) int {
	i := at + 2
	for i+1 < len(src) {
		if src[i] == '*' && src[i+1] == '/' {
			return i + 2 - at
		}
		i++
	}
	return 0
}
