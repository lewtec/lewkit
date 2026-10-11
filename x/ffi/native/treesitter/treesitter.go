// Package treesitter binds libtree-sitter and an installed grammar
// library (libtree-sitter-<name>) without cgo.
//
// TSNode matches tree-sitter's public struct: four uint32 context
// words, then the node id, then the tree pointer.
package treesitter

//go:generate go run ../../../../cmd/lewkit generate header --package treesitter --bind TreeSitter --out ts_bind.go ts.h
//go:generate go run ../../../../cmd/lewkit generate header --package treesitter --bind Libc --out libc_bind.go libc.h

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// Language is a TSLanguage pointer from a grammar shared library.
type Language uintptr

var errUnavailable = errors.New("tree-sitter library unavailable")

// runtimeLibs is the tree-sitter runtime and the libc that frees its strings.
type runtimeLibs struct {
	ts   TreeSitter
	libc Libc
}

var (
	bindTreeSitter = native.BindOnce[TreeSitter](native.Now|native.Global, runtimeSonames()...)
	bindLibc       = native.BindOnce[Libc](native.Now|native.Global, libcSoname())
)

// loadRuntime binds both libraries and returns them from that same call.
func loadRuntime() (runtimeLibs, error) {
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		return runtimeLibs{}, fmt.Errorf("%w: %s", errUnavailable, runtime.GOOS)
	}
	ts, err := bindTreeSitter()
	if err != nil {
		return runtimeLibs{}, err
	}
	libc, err := bindLibc()
	if err != nil {
		return runtimeLibs{}, err
	}
	return runtimeLibs{ts: ts, libc: libc}, nil
}

// Available loads libtree-sitter. Each library bind keeps its first result.
func Available() error {
	_, err := loadRuntime()
	return err
}

var (
	langMu sync.Mutex
	langs  = map[string]Language{}
)

// Languages returns grammar names found as libtree-sitter-<name> in the
// loader search path. The list is sorted and has no duplicates.
func Languages() []string {
	seen := map[string]struct{}{}
	for _, dir := range native.SearchDirs() {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			name, ok := grammarName(entry.Name())
			if !ok {
				continue
			}
			seen[name] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	slices.Sort(out)
	return out
}

// OpenLanguage loads libtree-sitter-<name> and its tree_sitter_<name> symbol.
func OpenLanguage(name string) (Language, error) {
	if err := Available(); err != nil {
		return 0, err
	}
	langMu.Lock()
	defer langMu.Unlock()
	if lang, ok := langs[name]; ok {
		return lang, nil
	}
	lib, err := native.OpenChain(native.Now|native.Global, grammarSonames(name)...)
	if err != nil {
		return 0, err
	}
	sym, err := native.Symbol(lib, symbolName(name))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", symbolName(name), err)
	}
	addr := callLanguage(sym)
	if addr == 0 {
		return 0, fmt.Errorf("%s: %w", name, errUnavailable)
	}
	lang := Language(addr)
	langs[name] = lang
	return lang, nil
}

// Parser is one native parser. Close is optional.
type Parser struct {
	mu   sync.Mutex
	p    uintptr
	libs runtimeLibs
}

// NewParser returns a parser loaded from the same call that opened the runtime.
func NewParser() (*Parser, error) {
	libs, err := loadRuntime()
	if err != nil {
		return nil, err
	}
	p := libs.ts.TsParserNew()
	if p == 0 {
		return nil, errUnavailable
	}
	out := &Parser{p: p, libs: libs}
	runtime.SetFinalizer(out, func(p *Parser) { p.Close() })
	return out, nil
}

// SetLanguage selects lang.
func (p *Parser) SetLanguage(lang Language) bool {
	if p == nil || p.p == 0 || lang == 0 {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.libs.ts.TsParserSetLanguage(p.p, uintptr(lang))
}

// Parse parses source. The returned tree keeps the native allocation alive.
func (p *Parser) Parse(source []byte) (*Tree, error) {
	if p == nil || p.p == 0 {
		return nil, errUnavailable
	}
	buf := make([]byte, len(source)+1)
	copy(buf, source)
	p.mu.Lock()
	defer p.mu.Unlock()
	raw := p.libs.ts.TsParserParseString(p.p, 0, &buf[0], uint32(len(source)))
	runtime.KeepAlive(buf)
	if raw == 0 {
		return nil, errUnavailable
	}
	tree := &Tree{raw: raw, libs: p.libs}
	runtime.SetFinalizer(tree, func(t *Tree) { t.Close() })
	return tree, nil
}

// Close frees the parser.
func (p *Parser) Close() {
	if p == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.p == 0 {
		return
	}
	p.libs.ts.TsParserDelete(p.p)
	p.p = 0
	runtime.SetFinalizer(p, nil)
}

// Tree is a native syntax tree.
type Tree struct {
	mu   sync.Mutex
	raw  uintptr
	libs runtimeLibs
}

// Root returns the root node. The node keeps t reachable.
func (t *Tree) Root() Node {
	if t == nil || t.raw == 0 {
		return Node{}
	}
	return Node{raw: t.libs.ts.TsTreeRootNode(t.raw), tree: t, libs: t.libs}
}

// Close frees the tree.
func (t *Tree) Close() {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.raw == 0 {
		return
	}
	t.libs.ts.TsTreeDelete(t.raw)
	t.raw = 0
	runtime.SetFinalizer(t, nil)
}

// Node is one syntax node. tree keeps the native tree alive.
// libs is the runtime that produced the node.
type Node struct {
	raw  RawNode
	tree *Tree
	libs runtimeLibs
}

func (n Node) live() bool {
	return n.tree != nil && n.raw.ID != 0 && !n.libs.ts.TsNodeIsNull(n.raw)
}

// Type returns the node type.
func (n Node) Type() string {
	if !n.live() {
		return ""
	}
	return native.GoString(n.libs.ts.TsNodeType(n.raw))
}

// StartByte returns the start offset.
func (n Node) StartByte() uint32 {
	if !n.live() {
		return 0
	}
	return n.libs.ts.TsNodeStartByte(n.raw)
}

// EndByte returns the end offset.
func (n Node) EndByte() uint32 {
	if !n.live() {
		return 0
	}
	return n.libs.ts.TsNodeEndByte(n.raw)
}

// ChildCount returns the number of children.
func (n Node) ChildCount() uint32 {
	if !n.live() {
		return 0
	}
	return n.libs.ts.TsNodeChildCount(n.raw)
}

// Child returns the child at index.
func (n Node) Child(index uint32) Node {
	if !n.live() {
		return Node{tree: n.tree, libs: n.libs}
	}
	return Node{raw: n.libs.ts.TsNodeChild(n.raw, index), tree: n.tree, libs: n.libs}
}

// FieldNameForChild returns the field name of the child at index.
func (n Node) FieldNameForChild(index uint32) string {
	if !n.live() {
		return ""
	}
	return native.GoString(n.libs.ts.TsNodeFieldNameForChild(n.raw, index))
}

// NamedChildCount returns the number of named children.
func (n Node) NamedChildCount() uint32 {
	if !n.live() {
		return 0
	}
	return n.libs.ts.TsNodeNamedChildCount(n.raw)
}

// NamedChild returns the named child at index.
func (n Node) NamedChild(index uint32) Node {
	if !n.live() {
		return Node{tree: n.tree, libs: n.libs}
	}
	return Node{raw: n.libs.ts.TsNodeNamedChild(n.raw, index), tree: n.tree, libs: n.libs}
}

// IsNull reports a null node.
func (n Node) IsNull() bool { return !n.live() }

// IsNamed reports a named node.
func (n Node) IsNamed() bool { return n.live() && n.libs.ts.TsNodeIsNamed(n.raw) }

// IsExtra reports an extra node.
func (n Node) IsExtra() bool { return n.live() && n.libs.ts.TsNodeIsExtra(n.raw) }

// IsError reports an error node.
func (n Node) IsError() bool { return n.live() && n.libs.ts.TsNodeIsError(n.raw) }

// HasError reports an error in this node or a descendant.
func (n Node) HasError() bool { return n.live() && n.libs.ts.TsNodeHasError(n.raw) }

// String returns the s-expression. The native buffer is freed.
func (n Node) String() string {
	if !n.live() {
		return ""
	}
	p := n.libs.ts.TsNodeString(n.raw)
	if p == 0 {
		return ""
	}
	s := native.GoString(p)
	n.libs.libc.Free(p)
	return s
}

func callLanguage(sym uintptr) uintptr {
	var ctor func() uintptr
	native.Register(&ctor, sym)
	return ctor()
}

func runtimeSonames() []string {
	if runtime.GOOS == "darwin" {
		return []string{"libtree-sitter.dylib", "libtree-sitter.0.dylib"}
	}
	return []string{"libtree-sitter.so.0", "libtree-sitter.so"}
}

func libcSoname() string {
	if runtime.GOOS == "darwin" {
		return "libSystem.B.dylib"
	}
	return "libc.so.6"
}

func grammarSonames(name string) []string {
	base := "libtree-sitter-" + name
	if runtime.GOOS == "darwin" {
		return []string{base + ".dylib"}
	}
	return []string{base + ".so", base + ".so.0"}
}

func symbolName(name string) string {
	return "tree_sitter_" + strings.ReplaceAll(name, "-", "_")
}

// grammarName returns the language name from a libtree-sitter-<name> file.
func grammarName(file string) (string, bool) {
	rest, ok := strings.CutPrefix(filepath.Base(file), "libtree-sitter-")
	if !ok || rest == "" {
		return "", false
	}
	for _, ext := range []string{".so", ".dylib"} {
		i := strings.Index(rest, ext)
		if i > 0 {
			return rest[:i], true
		}
	}
	return "", false
}
