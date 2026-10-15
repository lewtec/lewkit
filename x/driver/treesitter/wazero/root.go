// Package wazero registers the wazero tree-sitter engine.
//
// This package imports the wazero grammar runtime only.
// Blank-import the language modules you need.
package wazero

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	"github.com/lewtec/wazero-tree-sitter/grammar"
)

type factory struct{}

func (factory) ID() string                               { return "treesitter_wazero" }
func (factory) Name() string                             { return "wazero" }
func (factory) Weight() int                              { return 50 }
func (factory) CheckCompatibility(context.Context) error { return nil }
func (factory) New(context.Context) (treesitter.Driver, error) {
	return engine{}, nil
}

type engine struct{}

func (engine) Names() []string { return grammar.List() }

func (engine) Language(name string) (treesitter.Language, bool) {
	raw, ok := grammar.Get(name)
	return treesitter.Lookup(raw, ok, func(raw *grammar.Language) treesitter.Language {
		return lang{name: name, raw: raw}
	})
}

func (engine) ByExtension(filename string) (treesitter.Language, bool) {
	raw, ok := grammar.GetByExtension(filename)
	return treesitter.Lookup(raw, ok, func(raw *grammar.Language) treesitter.Language {
		return lang{name: nameOf(raw), raw: raw}
	})
}

func nameOf(raw *grammar.Language) string {
	return treesitter.NameOf(raw, grammar.List(), grammar.Get)
}

type lang struct {
	name string
	raw  *grammar.Language
}

func (l lang) Name() string { return l.name }

func (l lang) Parse(source []byte) (*treesitter.Tree, error) {
	return treesitter.Run(l.name, l.raw, source, grammar.NewParser, func(tree *grammar.Tree) treesitter.Node {
		return treesitter.NewSyntax[grammar.Node](tree, tree.RootNode())
	})
}

func init() { driver.Register[treesitter.Driver](factory{}) }
