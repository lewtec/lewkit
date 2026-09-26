// Package native parses with libtree-sitter and an installed grammar
// shared library.
//
// The factory is incompatible unless LEWKIT_ENABLE_NATIVE_TREESITTER
// is set. A set variable still fails when libtree-sitter is missing.
// Grammars are libtree-sitter-<name> on the loader path.
package native

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	tstree "github.com/lewtec/lewkit/x/ffi/native/treesitter"
)

type factory struct{}

func (factory) ID() string   { return "treesitter_native" }
func (factory) Name() string { return "tree-sitter" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	if err := driver.RequireEnv(ctx, "LEWKIT_ENABLE_NATIVE_TREESITTER"); err != nil {
		return err
	}
	if err := tstree.Available(); err != nil {
		return fmt.Errorf("%w: %s", driver.ErrIncompatible, err)
	}
	return nil
}

func (factory) New(context.Context) (treesitter.Driver, error) {
	return engine{}, nil
}

type engine struct{}

func (engine) Names() []string { return tstree.Languages() }

func (engine) Language(name string) (treesitter.Language, bool) {
	raw, err := tstree.OpenLanguage(name)
	if err != nil || raw == 0 {
		return nil, false
	}
	return lang{name: name, raw: raw}, true
}

func (engine) ByExtension(filename string) (treesitter.Language, bool) {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if ext == "" {
		return nil, false
	}
	if name, ok := extAlias[ext]; ok {
		if lang, ok := (engine{}).Language(name); ok {
			return lang, true
		}
	}
	return (engine{}).Language(ext)
}

var extAlias = map[string]string{
	"js":  "javascript",
	"mjs": "javascript",
	"cjs": "javascript",
	"jsx": "javascript",
	"ts":  "typescript",
	"py":  "python",
	"rb":  "ruby",
	"rs":  "rust",
	"cc":  "cpp",
	"cxx": "cpp",
	"hpp": "cpp",
	"h":   "c",
	"yml": "yaml",
	"md":  "markdown",
	"sh":  "bash",
}

type lang struct {
	name string
	raw  tstree.Language
}

func (l lang) Name() string { return l.name }

func (l lang) Parse(source []byte) (*treesitter.Tree, error) {
	parser, err := tstree.NewParser()
	if err != nil {
		return nil, err
	}
	defer parser.Close()
	if !parser.SetLanguage(l.raw) {
		return nil, fmt.Errorf("%w: %s", treesitter.ErrParse, l.name)
	}
	tree, err := parser.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", treesitter.ErrParse, l.name)
	}
	return treesitter.NewTree(syntax{n: tree.Root()}), nil
}

type syntax struct {
	n tstree.Node
}

func (s syntax) Type() string {
	return s.n.Type()
}

func (s syntax) StartByte() uint32 { return s.n.StartByte() }
func (s syntax) EndByte() uint32   { return s.n.EndByte() }

func (s syntax) ChildCount() uint32 { return s.n.ChildCount() }

func (s syntax) Child(index uint32) treesitter.Node {
	return syntax{n: s.n.Child(index)}
}

func (s syntax) NamedChildCount() uint32 { return s.n.NamedChildCount() }

func (s syntax) NamedChild(index uint32) treesitter.Node {
	return syntax{n: s.n.NamedChild(index)}
}

func (s syntax) FieldNameForChild(index uint32) string {
	return s.n.FieldNameForChild(index)
}

func (s syntax) IsNull() bool   { return s.n.IsNull() }
func (s syntax) IsNamed() bool  { return s.n.IsNamed() }
func (s syntax) IsExtra() bool  { return s.n.IsExtra() }
func (s syntax) IsError() bool  { return s.n.IsError() }
func (s syntax) HasError() bool { return s.n.HasError() }
func (s syntax) String() string { return s.n.String() }

func init() { driver.Register[treesitter.Driver](factory{}) }
