//go:build !windows

// Package ccgo registers the ccgo tree-sitter engine.
//
// This package imports the ccgo grammar runtime only.
// Blank-import the language modules you need.
package ccgo

import (
	"context"
	"fmt"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	grammar "github.com/modernc-tree-sitter/ccgo-tree-sitter/core"
)

type factory struct{}

func (factory) ID() string                               { return "treesitter_ccgo" }
func (factory) Name() string                             { return "ccgo" }
func (factory) Weight() int                              { return 40 }
func (factory) CheckCompatibility(context.Context) error { return nil }
func (factory) New(context.Context) (treesitter.Driver, error) {
	return engine{}, nil
}

type engine struct{}

func (engine) Names() []string { return grammar.List() }

func (engine) Language(name string) (treesitter.Language, bool) {
	raw, ok := grammar.Get(name)
	if !ok || raw == nil {
		return nil, false
	}
	return lang{name: name, raw: raw}, true
}

func (engine) ByExtension(filename string) (treesitter.Language, bool) {
	raw, ok := grammar.GetByExtension(filename)
	if !ok || raw == nil {
		return nil, false
	}
	return lang{name: nameOf(raw), raw: raw}, true
}

func nameOf(raw grammar.Language) string {
	return treesitter.NameOf(raw, grammar.List(), grammar.Get)
}

type lang struct {
	name string
	raw  grammar.Language
}

func (l lang) Name() string { return l.name }

func (l lang) Parse(source []byte) (*treesitter.Tree, error) {
	parser := grammar.NewParser()
	defer parser.Delete()
	if !parser.SetLanguage(l.raw) {
		return nil, fmt.Errorf("%w: %s", treesitter.ErrParse, l.name)
	}
	tree := parser.ParseBytes(source)
	if tree == nil {
		return nil, fmt.Errorf("%w: %s", treesitter.ErrParse, l.name)
	}
	return treesitter.NewTree(treesitter.NewSyntax[grammar.Node](tree, tree.RootNode())), nil
}

func init() { driver.Register[treesitter.Driver](factory{}) }
