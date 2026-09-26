// Package treesitter binds libtree-sitter and an installed grammar
// library (libtree-sitter-<name>) without cgo.
//
// TSNode matches tree-sitter's public struct: four uint32 context
// words, then the node id, then the tree pointer.
package treesitter

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

// RawNode is one tree-sitter TSNode.
type RawNode struct {
	Context [4]uint32
	ID      uintptr
	Tree    uintptr
}

// Language is a TSLanguage pointer from a grammar shared library.
type Language uintptr

var (
	errUnavailable = errors.New("tree-sitter library unavailable")

	loadOnce sync.Once
	loadErr  error

	parserNew    func() uintptr
	parserDelete func(uintptr)
	setLanguage  func(parser uintptr, lang uintptr) bool
	parseString  func(parser uintptr, old uintptr, str *byte, length uint32) uintptr
	treeDelete   func(uintptr)
	rootNode     func(tree uintptr) RawNode

	nodeType       func(RawNode) uintptr
	nodeStartByte  func(RawNode) uint32
	nodeEndByte    func(RawNode) uint32
	nodeChildCount func(RawNode) uint32
	nodeChild      func(RawNode, uint32) RawNode
	nodeFieldName  func(RawNode, uint32) uintptr
	nodeNamedCount func(RawNode) uint32
	nodeNamedChild func(RawNode, uint32) RawNode
	nodeString     func(RawNode) uintptr
	nodeIsNull     func(RawNode) bool
	nodeIsNamed    func(RawNode) bool
	nodeIsExtra    func(RawNode) bool
	nodeIsError    func(RawNode) bool
	nodeHasError   func(RawNode) bool
	libcFree       func(uintptr)
)

// Available loads libtree-sitter. Later calls return the first result.
func Available() error {
	switch runtime.GOOS {
	case "linux", "darwin":
	default:
		return fmt.Errorf("%w: %s", errUnavailable, runtime.GOOS)
	}
	loadOnce.Do(func() { loadErr = bindRuntime() })
	return loadErr
}

func bindRuntime() error {
	lib, err := openFirst(runtimeSonames())
	if err != nil {
		return err
	}
	libc, err := openFirst([]string{libcSoname()})
	if err != nil {
		return err
	}
	binds := []struct {
		lib  uintptr
		name string
		fn   any
	}{
		{lib, "ts_parser_new", &parserNew},
		{lib, "ts_parser_delete", &parserDelete},
		{lib, "ts_parser_set_language", &setLanguage},
		{lib, "ts_parser_parse_string", &parseString},
		{lib, "ts_tree_delete", &treeDelete},
		{lib, "ts_tree_root_node", &rootNode},
		{lib, "ts_node_type", &nodeType},
		{lib, "ts_node_start_byte", &nodeStartByte},
		{lib, "ts_node_end_byte", &nodeEndByte},
		{lib, "ts_node_child_count", &nodeChildCount},
		{lib, "ts_node_child", &nodeChild},
		{lib, "ts_node_field_name_for_child", &nodeFieldName},
		{lib, "ts_node_named_child_count", &nodeNamedCount},
		{lib, "ts_node_named_child", &nodeNamedChild},
		{lib, "ts_node_string", &nodeString},
		{lib, "ts_node_is_null", &nodeIsNull},
		{lib, "ts_node_is_named", &nodeIsNamed},
		{lib, "ts_node_is_extra", &nodeIsExtra},
		{lib, "ts_node_is_error", &nodeIsError},
		{lib, "ts_node_has_error", &nodeHasError},
		{libc, "free", &libcFree},
	}
	for _, item := range binds {
		if err := bind(item.lib, item.name, item.fn); err != nil {
			return err
		}
	}
	return nil
}

var (
	langMu sync.Mutex
	langs  = map[string]Language{}
)

// Languages returns grammar names found as libtree-sitter-<name> in the
// loader search path. The list is sorted and has no duplicates.
func Languages() []string {
	seen := map[string]struct{}{}
	for _, dir := range libDirs() {
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
	lib, err := openFirst(grammarSonames(name))
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
	mu sync.Mutex
	p  uintptr
}

// NewParser returns a parser. The runtime library must already be available.
func NewParser() (*Parser, error) {
	if err := Available(); err != nil {
		return nil, err
	}
	p := parserNew()
	if p == 0 {
		return nil, errUnavailable
	}
	out := &Parser{p: p}
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
	return setLanguage(p.p, uintptr(lang))
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
	raw := parseString(p.p, 0, &buf[0], uint32(len(source)))
	runtime.KeepAlive(buf)
	if raw == 0 {
		return nil, errUnavailable
	}
	tree := &Tree{raw: raw}
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
	parserDelete(p.p)
	p.p = 0
	runtime.SetFinalizer(p, nil)
}

// Tree is a native syntax tree.
type Tree struct {
	mu  sync.Mutex
	raw uintptr
}

// Root returns the root node. The node keeps t reachable.
func (t *Tree) Root() Node {
	if t == nil || t.raw == 0 {
		return Node{}
	}
	return Node{raw: rootNode(t.raw), tree: t}
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
	treeDelete(t.raw)
	t.raw = 0
	runtime.SetFinalizer(t, nil)
}

// Node is one syntax node. tree keeps the native tree alive.
type Node struct {
	raw  RawNode
	tree *Tree
}

func (n Node) live() bool {
	return n.tree != nil && n.raw.ID != 0 && !nodeIsNull(n.raw)
}

// Type returns the node type.
func (n Node) Type() string {
	if !n.live() {
		return ""
	}
	return goString(nodeType(n.raw))
}

// StartByte returns the start offset.
func (n Node) StartByte() uint32 {
	if !n.live() {
		return 0
	}
	return nodeStartByte(n.raw)
}

// EndByte returns the end offset.
func (n Node) EndByte() uint32 {
	if !n.live() {
		return 0
	}
	return nodeEndByte(n.raw)
}

// ChildCount returns the number of children.
func (n Node) ChildCount() uint32 {
	if !n.live() {
		return 0
	}
	return nodeChildCount(n.raw)
}

// Child returns the child at index.
func (n Node) Child(index uint32) Node {
	if !n.live() {
		return Node{tree: n.tree}
	}
	return Node{raw: nodeChild(n.raw, index), tree: n.tree}
}

// FieldNameForChild returns the field name of the child at index.
func (n Node) FieldNameForChild(index uint32) string {
	if !n.live() {
		return ""
	}
	return goString(nodeFieldName(n.raw, index))
}

// NamedChildCount returns the number of named children.
func (n Node) NamedChildCount() uint32 {
	if !n.live() {
		return 0
	}
	return nodeNamedCount(n.raw)
}

// NamedChild returns the named child at index.
func (n Node) NamedChild(index uint32) Node {
	if !n.live() {
		return Node{tree: n.tree}
	}
	return Node{raw: nodeNamedChild(n.raw, index), tree: n.tree}
}

// IsNull reports a null node.
func (n Node) IsNull() bool { return !n.live() }

// IsNamed reports a named node.
func (n Node) IsNamed() bool { return n.live() && nodeIsNamed(n.raw) }

// IsExtra reports an extra node.
func (n Node) IsExtra() bool { return n.live() && nodeIsExtra(n.raw) }

// IsError reports an error node.
func (n Node) IsError() bool { return n.live() && nodeIsError(n.raw) }

// HasError reports an error in this node or a descendant.
func (n Node) HasError() bool { return n.live() && nodeHasError(n.raw) }

// String returns the s-expression. The native buffer is freed.
func (n Node) String() string {
	if !n.live() {
		return ""
	}
	p := nodeString(n.raw)
	if p == 0 {
		return ""
	}
	s := goString(p)
	libcFree(p)
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

func openFirst(sonames []string) (uintptr, error) {
	var last error
	for _, soname := range sonames {
		lib, err := openLib(soname)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	if last == nil {
		last = errUnavailable
	}
	return 0, last
}

func openLib(soname string) (uintptr, error) {
	var last error
	for _, path := range libPaths(soname) {
		lib, err := native.Open(path, native.Now|native.Global)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	if last == nil {
		last = errUnavailable
	}
	return 0, fmt.Errorf("%s: %w", soname, last)
}

func libPaths(soname string) []string {
	paths := []string{soname}
	for _, dir := range libDirs() {
		if dir == "" {
			continue
		}
		paths = append(paths, filepath.Join(dir, soname))
	}
	return paths
}

func libDirs() []string {
	var dirs []string
	if dir := os.Getenv("LEWKIT_LIB"); dir != "" {
		dirs = append(dirs, dir)
	}
	if list := os.Getenv("LD_LIBRARY_PATH"); list != "" {
		dirs = append(dirs, strings.Split(list, string(os.PathListSeparator))...)
	}
	dirs = append(dirs,
		"/run/current-system/sw/lib",
		"/usr/lib",
		"/usr/lib64",
		"/usr/local/lib",
	)
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".nix-profile/lib"))
	}
	return dirs
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

func bind(lib uintptr, name string, fnptr any) error {
	if _, err := native.Symbol(lib, name); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	native.Func(lib, name, fnptr)
	return nil
}

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
		if n > 1<<20 {
			break
		}
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}
