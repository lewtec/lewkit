package parse

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type patTerm string

const (
	patKey   patTerm = "key"
	patEqual patTerm = "="
	patWord  patTerm = "word"
	patSpace patTerm = "space"
	patNum   patTerm = "number"
	patPlus  patTerm = "+"
)

func (term patTerm) String() string { return string(term) }

func (term patTerm) IsValid() bool {
	switch term {
	case patKey, patEqual, patWord, patSpace, patNum, patPlus:
		return true
	default:
		return false
	}
}

type patRule string

const (
	patFile patRule = "file"
	patPair patRule = "pair"
)

func (rule patRule) String() string { return string(rule) }

func (rule patRule) IsValid() bool { return rule == patFile || rule == patPair }

func wordByteTest(c byte, first bool) bool {
	if c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_' {
		return true
	}
	return !first && c >= '0' && c <= '9'
}

func TestPatternAfterSkipsSpace(t *testing.T) {
	var syntax Syntax[patTerm, patRule]
	key := syntax.Pat(patKey, Class(wordByteTest).Unless(After(patEqual)))
	word := syntax.Pat(patWord, Class(wordByteTest))
	equal := syntax.Pat(patEqual, Byte('='))
	space := syntax.Pat(patSpace, While(" \t"))
	grammar := syntax.Compose(
		syntax.Eq(patFile, syntax.Repeat(syntax.Alt(key, equal, word, space))),
		syntax.Start(patFile),
	)
	model, err := syntax.Compile("pattern", grammar)
	require.NoError(t, err)
	root, err := model.Parse([]byte("a =b c"))
	require.NoError(t, err)
	got := make([]patTerm, 0, root.ChildCount())
	for i := 0; i < root.ChildCount(); i++ {
		term, ok := root.Child(i).Term()
		require.True(t, ok)
		got = append(got, term)
	}
	require.Equal(t, []patTerm{patKey, patSpace, patEqual, patWord, patSpace, patKey}, got)
}

func TestPatternKeywordLooksAhead(t *testing.T) {
	var syntax Syntax[patTerm, patRule]
	export := syntax.Pat(patKey, Keyword("export").Look(While(" \t")))
	word := syntax.Pat(patWord, Class(wordByteTest))
	space := syntax.Pat(patSpace, While(" \t"))
	grammar := syntax.Compose(
		syntax.Eq(patFile, syntax.Repeat(syntax.Alt(export, word, space))),
		syntax.Start(patFile),
	)
	model, err := syntax.Compile("keyword", grammar)
	require.NoError(t, err)
	root, err := model.Parse([]byte("export1 export x"))
	require.NoError(t, err)
	got := make([]patTerm, 0, root.ChildCount())
	for i := 0; i < root.ChildCount(); i++ {
		term, ok := root.Child(i).Term()
		require.True(t, ok)
		if term == patSpace {
			continue
		}
		got = append(got, term)
	}
	require.Equal(t, []patTerm{patWord, patKey, patWord}, got)
}

func TestRecordFillAndTable(t *testing.T) {
	var syntax Syntax[patTerm, patRule]
	number := syntax.Pat(patNum, Class(func(c byte, _ bool) bool { return c >= '0' && c <= '9' }))
	plus := syntax.Pat(patPlus, Byte('+'))
	pair := syntax.Eq(patPair, syntax.Seq(
		syntax.Text(number, "Left"),
		syntax.Skip(plus),
		syntax.Text(number, "Right"),
	))
	file := syntax.Eq(patFile, syntax.Seq(syntax.Ref(patPair))).Build(func(node SyntaxNode[patTerm, patRule]) (any, error) {
		var row struct {
			Left  string
			Right string
		}
		if err := node.Child(0).Fill(&row); err != nil {
			return nil, err
		}
		return row.Left + row.Right, nil
	})
	piece := syntax.Compose(file, pair, syntax.Start(patFile))
	prepared, err := syntax.Prepare("pattern", piece)
	require.NoError(t, err)
	compiled, err := prepared.Compile()
	require.NoError(t, err)
	table, err := compiled.Table()
	require.NoError(t, err)
	loaded, err := prepared.Load(table)
	require.NoError(t, err)
	again, err := loaded.Table()
	require.NoError(t, err)
	require.Equal(t, table, again)

	root, err := loaded.Parse([]byte("1+2"))
	require.NoError(t, err)
	got, err := root.Apply()
	require.NoError(t, err)
	require.Equal(t, "12", got)

	fork := syntax.Compose(
		syntax.Eq(patFile, syntax.Alt(
			syntax.Seq(syntax.Ref(patFile), plus, syntax.Ref(patFile)),
			number,
		)),
		syntax.Start(patFile),
	)
	forkPrep, err := syntax.Prepare("fork", fork)
	require.NoError(t, err)
	forkModel, err := forkPrep.Compile()
	require.NoError(t, err)
	forkTable, err := forkModel.Table()
	require.NoError(t, err)
	forkLoad, err := forkPrep.Load(forkTable)
	require.NoError(t, err)
	require.Equal(t, forkModel.language.States(), forkLoad.language.States())
	require.Equal(t, forkModel.language.Forks(), forkLoad.language.Forks())
	left, err := forkModel.language.Parse([]byte("1+2+3"))
	require.NoError(t, err)
	right, err := forkLoad.language.Parse([]byte("1+2+3"))
	require.NoError(t, err)
	require.Equal(t, left.SExpr(), right.SExpr())
}

type modeTerm string

const (
	modeTermRule   modeTerm = "rule"
	modeTermDefine modeTerm = "define"
	modeTermEndef  modeTerm = "endef"
	modeTermAssign modeTerm = "assign"
	modeTermRecipe modeTerm = "recipe"
)

func (term modeTerm) String() string { return string(term) }

func TestModeDepth(t *testing.T) {
	const (
		modeRecipe Mode = iota + 1
		modeDefine
	)
	modes := Modes[modeTerm]{
		{On: modeTermRule, Enter: modeRecipe},
		{On: modeTermAssign, Leave: modeRecipe},
		{On: modeTermDefine, Enter: modeDefine, Depth: true},
		{On: modeTermDefine, Leave: modeRecipe},
		{On: modeTermEndef, Leave: modeDefine, Depth: true},
		{In: modeRecipe, Not: modeDefine, When: Byte('\t'), Emit: modeTermRecipe},
	}
	st := &LexState{}
	mem := Remember(st, modes)
	mem.Note(modeTermRule)
	require.True(t, mem.Active(modeRecipe))
	term, _, end, matched, ok := mem.lead([]byte("\tline"), 0, st)
	require.True(t, matched)
	require.True(t, ok)
	require.Equal(t, modeTermRecipe, term)
	require.Equal(t, 1, end)
	mem.Note(modeTermDefine)
	require.False(t, mem.Active(modeRecipe))
	require.Equal(t, 1, mem.Depth(modeDefine))
	mem.Enter(modeDefine, true)
	require.Equal(t, 2, mem.Depth(modeDefine))
	mem.Leave(modeDefine, true)
	mem.Note(modeTermEndef)
	require.Equal(t, 0, mem.Depth(modeDefine))
	_, _, _, matched, _ = mem.lead([]byte("\tline"), 0, st)
	require.False(t, matched)
}
