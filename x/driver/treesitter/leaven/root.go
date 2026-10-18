// Package leaven registers the leaven tree-sitter engine.
//
// Blank-import a language package from
// github.com/lewtec/leaven-tree-sitter/grammar/<name>
// so Get on this engine can see it.
package leaven

import (
	"context"
	"fmt"

	"github.com/lewtec/leaven-tree-sitter/grammar"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/treesitter"
)

type factory struct{}

func (factory) ID() string                               { return "treesitter_leaven" }
func (factory) Name() string                             { return "leaven" }
func (factory) Weight() int                              { return 60 }
func (factory) CheckCompatibility(context.Context) error { return nil }
func (factory) New(context.Context) (treesitter.Driver, error) {
	return engine{}, nil
}

type engine struct{}

func (engine) Names() []string { return grammar.List() }

func (engine) Language(name string) (treesitter.Language, bool) {
	raw, ok := grammar.Get(name)
	return treesitter.Lookup(raw, ok, func(raw grammar.Language) treesitter.Language {
		return lang{name: name, raw: raw}
	})
}

func (engine) ByExtension(filename string) (treesitter.Language, bool) {
	raw, ok := grammar.GetByExtension(filename)
	return treesitter.Lookup(raw, ok, func(raw grammar.Language) treesitter.Language {
		return lang{name: nameOf(raw), raw: raw}
	})
}

func nameOf(raw grammar.Language) string {
	return treesitter.NameOf(raw, grammar.List(), grammar.Get)
}

type lang struct {
	name string
	raw  grammar.Language
}

func (l lang) Name() string { return l.name }

func (l lang) Parse(source []byte) (tree *treesitter.Tree, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			tree = nil
			err = fmt.Errorf("%w: %s: lexer failed", treesitter.ErrParse, l.name)
		}
	}()
	return treesitter.Run(l.name, l.raw, source, newPinnedParser, func(tree *grammar.Tree) treesitter.Node {
		return treesitter.NewSyntax[grammar.Node](tree, tree.RootNode())
	})
}

func init() { driver.Register[treesitter.Driver](factory{}) }
