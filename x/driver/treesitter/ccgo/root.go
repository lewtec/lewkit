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
	"github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar"
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
	for _, name := range grammar.List() {
		got, ok := grammar.Get(name)
		if ok && got == raw {
			return name
		}
	}
	return ""
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
	return treesitter.NewTree(syntax{tree: tree, n: tree.RootNode()}), nil
}

type syntax struct {
	tree *grammar.Tree
	n    *grammar.Node
}

func (s syntax) live() bool { return s.n != nil && !s.n.IsNull() }

func (s syntax) Type() string {
	if !s.live() {
		return ""
	}
	return s.n.Type()
}

func (s syntax) StartByte() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.StartByte()
}

func (s syntax) EndByte() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.EndByte()
}

func (s syntax) ChildCount() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.ChildCount()
}

func (s syntax) Child(index uint32) treesitter.Node {
	if !s.live() {
		return syntax{tree: s.tree}
	}
	return syntax{tree: s.tree, n: s.n.Child(index)}
}

func (s syntax) NamedChildCount() uint32 {
	if !s.live() {
		return 0
	}
	return s.n.NamedChildCount()
}

func (s syntax) NamedChild(index uint32) treesitter.Node {
	if !s.live() {
		return syntax{tree: s.tree}
	}
	return syntax{tree: s.tree, n: s.n.NamedChild(index)}
}

func (s syntax) FieldNameForChild(index uint32) string {
	if !s.live() {
		return ""
	}
	return s.n.FieldNameForChild(index)
}

func (s syntax) IsNull() bool  { return !s.live() }
func (s syntax) IsNamed() bool { return s.live() && s.n.IsNamed() }
func (s syntax) IsExtra() bool { return s.live() && s.n.IsExtra() }
func (s syntax) IsError() bool { return s.live() && s.n.IsError() }
func (s syntax) HasError() bool {
	return s.live() && s.n.HasError()
}

func (s syntax) String() string {
	if !s.live() {
		return ""
	}
	return s.n.String()
}

func init() { driver.Register[treesitter.Driver](factory{}) }
