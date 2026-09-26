// Package treesitter parses source with tree-sitter.
//
// Blank-import the prelude to register the engines. Blank-import a
// language package so that name becomes visible here. When more than
// one engine has the name, the higher weight wins.
//
// Language modules require grammar v0.0.0. In the main module, replace
// that version with the grammar module version this driver uses.
//
//	import (
//		_ "github.com/lewtec/lewkit/x/driver/treesitter/prelude"
//		_ "github.com/lewtec/leaven-tree-sitter/grammar/json"
//	)
//
//	tree, err := treesitter.Parse(ctx, "json", []byte(`{"a":1}`))
package treesitter

import (
	"context"
	"errors"
	"maps"
	"slices"

	"github.com/lewtec/lewkit/x/driver"
)

var (
	// ErrUnknown means no registered engine has that language.
	ErrUnknown = errors.New("unknown tree-sitter language")
	// ErrParse means the engine rejected the language or returned no tree.
	ErrParse = errors.New("tree-sitter parse failed")
)

// Driver is one tree-sitter engine. Language reports only grammars that
// engine has registered.
type Driver interface {
	Names() []string
	Language(name string) (Language, bool)
	ByExtension(filename string) (Language, bool)
}

// Language parses source with one grammar.
type Language interface {
	Name() string
	Parse(source []byte) (*Tree, error)
}

// Node is one syntax node. A null node returns empty values.
type Node interface {
	Type() string
	StartByte() uint32
	EndByte() uint32
	ChildCount() uint32
	Child(index uint32) Node
	NamedChildCount() uint32
	NamedChild(index uint32) Node
	FieldNameForChild(index uint32) string
	IsNull() bool
	IsNamed() bool
	IsExtra() bool
	IsError() bool
	HasError() bool
	String() string
}

// Tree is a parse result. RootNode is nil when the tree is nil.
type Tree struct {
	root Node
}

// NewTree returns a tree whose root is node.
func NewTree(root Node) *Tree {
	return &Tree{root: root}
}

// RootNode returns the root, or nil when t is nil.
func (t *Tree) RootNode() Node {
	if t == nil {
		return nil
	}
	return t.root
}

// Get returns the grammar registered as name.
// Engines are tried highest weight first.
func Get(ctx context.Context, name string) (Language, error) {
	if name == "" {
		return nil, ErrUnknown
	}
	return find(ctx, func(d Driver) (Language, bool) {
		return d.Language(name)
	})
}

// ForFile returns the grammar for filename's extension.
func ForFile(ctx context.Context, filename string) (Language, error) {
	return find(ctx, func(d Driver) (Language, bool) {
		return d.ByExtension(filename)
	})
}

// Parse parses source with the grammar named name.
func Parse(ctx context.Context, name string, source []byte) (*Tree, error) {
	lang, err := Get(ctx, name)
	if err != nil {
		return nil, err
	}
	return lang.Parse(source)
}

// Names returns the sorted union of grammar names across engines.
func Names(ctx context.Context) ([]string, error) {
	handles, err := driver.List[Driver](ctx)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, handle := range handles {
		engine, err := handle.Open(ctx)
		if err != nil {
			continue
		}
		for _, name := range engine.Names() {
			seen[name] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(seen)), nil
}

func find(ctx context.Context, pick func(Driver) (Language, bool)) (Language, error) {
	handles, err := driver.List[Driver](ctx)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		engine, err := handle.Open(ctx)
		if err != nil {
			continue
		}
		lang, ok := pick(engine)
		if ok && lang != nil {
			return lang, nil
		}
	}
	return nil, ErrUnknown
}
