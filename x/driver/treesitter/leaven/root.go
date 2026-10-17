// Package leaven registers the leaven tree-sitter engine.
//
// Blank-import a language package from
// github.com/lewtec/leaven-tree-sitter/grammar/<name>
// so Get on this engine can see it.
package leaven

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"

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
	// The leaven parser keeps the source pointer in a structure the garbage
	// collector does not scan. A collection during the lex frees that
	// allocation and the next byte read faults in ts_decode_utf8.
	release := holdGC()
	defer release()
	return treesitter.Run(l.name, l.raw, source, grammar.NewParser, func(tree *grammar.Tree) treesitter.Node {
		return treesitter.NewSyntax[grammar.Node](tree, tree.RootNode())
	})
}

// gcHold counts overlapping parses so the collector stays off until the
// last one returns. SetGCPercent is process-wide.
var gcHold struct {
	mu   sync.Mutex
	n    int
	prev int
}

func holdGC() func() {
	gcHold.mu.Lock()
	if gcHold.n == 0 {
		gcHold.prev = debug.SetGCPercent(-1)
	}
	gcHold.n++
	gcHold.mu.Unlock()
	return func() {
		gcHold.mu.Lock()
		gcHold.n--
		if gcHold.n == 0 {
			debug.SetGCPercent(gcHold.prev)
		}
		gcHold.mu.Unlock()
	}
}

func init() { driver.Register[treesitter.Driver](factory{}) }
