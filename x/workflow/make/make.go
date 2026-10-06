// Package make reads a makefile into a workflow graph.
//
// The subset covers a small Linux build: assignments, conditionals,
// includes, pattern rules, automatic variables, and the usual text
// functions. Recipes are expanded here and run by workflow, so a
// conda-activated CC and PATH are the compiler the recipe invokes.
package make

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/workflow"
)

var (
	// ErrSyntax means the makefile could not be parsed.
	ErrSyntax = errors.New("syntax")
	// ErrNoRule means a target has no rule and no file.
	ErrNoRule = errors.New("no rule to make target")
	// ErrCycle means prerequisites depend on the target.
	ErrCycle = errors.New("cycle")
)

type origin int

const (
	originUndefined origin = iota
	originDefault
	originEnv
	originFile
	originCommand
	originOverride
	originAutomatic
)

type kind int

const (
	kindRecursive kind = iota
	kindSimple
	kindShell
	kindConditional
	kindAppend
)

type variable struct {
	value  string
	simple bool
	origin origin
}

type rule struct {
	prereqs []string
	order   []string
	recipe  []string
	stem    string
}

type pattern struct {
	target string
	body   *rule
}

// File is a parsed makefile. Load reads it. Graph resolves targets.
type File struct {
	dir        string
	path       string
	ctx        context.Context
	vars       map[string]*variable
	exported   map[string]bool
	unexported map[string]bool
	exportAll  bool
	oneshell   bool
	explicit   map[string]*rule
	patterns   []pattern
	targetVars []targetVar
	phonies    map[string]bool
	first      string
	included   map[string]bool
	missing    []missedInclude
}

// Load reads path. work is the directory recipes run in; an empty work
// uses the makefile's directory. assigns are command-line NAME=VALUE lines.
// A missing included makefile is remade from a rule, then the read starts over.
func Load(ctx context.Context, path, work string, assigns, goals []string) (*File, error) {
	return load(ctx, path, work, assigns, goals, 0)
}

const maxRemake = 3

func load(ctx context.Context, path, work string, assigns, goals []string, depth int) (*File, error) {
	if ctx == nil {
		return nil, errors.New("make: nil context")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if work == "" {
		work = filepath.Dir(abs)
	} else {
		work, err = filepath.Abs(work)
		if err != nil {
			return nil, err
		}
	}
	f := &File{
		dir:        work,
		path:       abs,
		ctx:        ctx,
		vars:       map[string]*variable{},
		exported:   map[string]bool{},
		unexported: map[string]bool{},
		explicit:   map[string]*rule{},
		phonies:    map[string]bool{},
		included:   map[string]bool{},
	}
	f.defaults()
	f.importEnv()
	if err := f.commandLine(assigns); err != nil {
		return nil, err
	}
	if len(goals) > 0 {
		f.vars["MAKECMDGOALS"] = &variable{value: strings.Join(goals, " "), simple: true, origin: originCommand}
	}
	if err := f.includeFile(ctx, abs, false, abs, "", 0); err != nil {
		return nil, err
	}
	f.addBuiltins()
	if len(f.missing) == 0 {
		return f, nil
	}
	created, err := f.remakeMissing(ctx)
	if err != nil {
		return nil, err
	}
	if created {
		if depth >= maxRemake {
			return nil, f.missingError()
		}
		return load(ctx, path, work, assigns, goals, depth+1)
	}
	// A required include is still absent. An optional one is ignored.
	if f.pendingRequired() {
		return nil, f.missingError()
	}
	return f, nil
}

func (f *File) pendingRequired() bool {
	for _, m := range f.missing {
		if m.optional {
			continue
		}
		if _, err := os.Stat(m.abs); err != nil {
			return true
		}
	}
	return false
}

func (f *File) defaults() {
	for name, value := range map[string]string{
		"CC":  "cc",
		"CXX": "c++",
		"AR":  "ar",
		"RM":  "rm -f",
	} {
		f.vars[name] = &variable{value: value, simple: true, origin: originDefault}
	}
	f.vars["CURDIR"] = &variable{value: f.dir, simple: true, origin: originFile}
	// Enough of GNU make 4 for the kernel's version gate. output-sync is the
	// feature that gate checks; the others are ones this package implements.
	f.vars[".FEATURES"] = &variable{value: "order-only else-if oneshell output-sync", simple: true, origin: originDefault}
	f.vars["MAKE_VERSION"] = &variable{value: "4.3", simple: true, origin: originDefault}
	f.vars["MAKECMDGOALS"] = &variable{value: "", simple: true, origin: originCommand}
	// $(MAKE) reinvokes this tool. The kernel recurses once when MAKEFLAGS
	// lacks --no-print-directory; starting with the flag is that second read.
	f.vars["MAKE"] = &variable{value: makeCommand(), simple: true, origin: originDefault}
	f.vars["MAKEFLAGS"] = &variable{value: "--no-print-directory", simple: true, origin: originDefault}
}

func makeCommand() string {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return "lewkit workflow make"
	}
	return exe + " workflow make"
}

func (f *File) importEnv() {
	for _, kv := range os.Environ() {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || !ident(key) {
			continue
		}
		switch key {
		case "CURDIR", ".FEATURES", "MAKE", "MAKEFLAGS", "MAKE_VERSION", "MAKECMDGOALS", "MAKEFILE_LIST":
			continue
		}
		f.vars[key] = &variable{value: value, simple: true, origin: originEnv}
	}
}

func (f *File) commandLine(assigns []string) error {
	for _, line := range assigns {
		if err := f.applyAssign(line, originCommand, f.path, 0); err != nil {
			return err
		}
		name, _, _, _, ok := splitAssign(line)
		if ok && name != "" {
			f.exported[name] = true
		}
	}
	return nil
}

func (f *File) addBuiltins() {
	specs := []struct{ target, pre, recipe string }{
		{"%.o", "%.c", "$(CC) $(CFLAGS) $(CPPFLAGS) -c -o $@ $<"},
		{"%.o", "%.cc", "$(CXX) $(CXXFLAGS) $(CPPFLAGS) -c -o $@ $<"},
		{"%.o", "%.cpp", "$(CXX) $(CXXFLAGS) $(CPPFLAGS) -c -o $@ $<"},
	}
	for _, spec := range specs {
		body := &rule{prereqs: []string{spec.pre}, recipe: []string{spec.recipe}}
		f.patterns = append(f.patterns, pattern{target: spec.target, body: body})
	}
}

// ParseArgs splits make arguments into assignments and targets.
func ParseArgs(args []string) (assigns, targets []string) {
	for _, arg := range args {
		if _, _, _, _, ok := splitAssign(arg); ok {
			assigns = append(assigns, arg)
			continue
		}
		targets = append(targets, arg)
	}
	return assigns, targets
}

func ident(name string) bool {
	if name == "" {
		return false
	}
	for i := 0; i < len(name); i++ {
		if !nameChar(name[i]) {
			return false
		}
	}
	return true
}

func nameChar(c byte) bool {
	return c == '_' || c == '.' || c == '-' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func splitAssign(line string) (name, op, value string, override bool, ok bool) {
	s := strings.TrimSpace(line)
	if s == "" {
		return "", "", "", false, false
	}
	if strings.HasPrefix(s, "override") && boundary(s, len("override")) {
		override = true
		s = strings.TrimSpace(s[len("override"):])
	}
	if strings.HasPrefix(s, "export") && boundary(s, len("export")) {
		s = strings.TrimSpace(s[len("export"):])
	}
	if strings.HasPrefix(s, "unexport") && boundary(s, len("unexport")) {
		return "", "", "", false, false
	}
	i := 0
	for i < len(s) && nameChar(s[i]) {
		i++
	}
	if i == 0 {
		return "", "", "", false, false
	}
	name = s[:i]
	rest := strings.TrimSpace(s[i:])
	if rest == "" {
		return "", "", "", false, false
	}
	for _, candidate := range []string{"::=", ":=", "!=", "?=", "+=", "="} {
		if strings.HasPrefix(rest, candidate) {
			return name, candidate, strings.TrimSpace(rest[len(candidate):]), override, true
		}
	}
	return "", "", "", false, false
}

func boundary(s string, n int) bool {
	return len(s) == n || s[n] == ' ' || s[n] == '\t'
}

func (f *File) assign(name, value, op string, origin origin) error {
	cur := f.vars[name]
	kind := kindOf(op)
	if kind != kindAppend && kind != kindConditional && cur != nil && cur.origin > origin {
		return nil
	}
	if kind == kindAppend && cur != nil && cur.origin > origin {
		return nil
	}
	switch kind {
	case kindRecursive:
		f.vars[name] = &variable{value: value, origin: origin}
	case kindSimple:
		expanded, err := f.expand(value, nil)
		if err != nil {
			return err
		}
		f.vars[name] = &variable{value: expanded, simple: true, origin: origin}
	case kindShell:
		script, err := f.expand(value, nil)
		if err != nil {
			return err
		}
		out, err := f.shell(script)
		if err != nil {
			return err
		}
		f.vars[name] = &variable{value: out, simple: true, origin: origin}
	case kindConditional:
		if cur != nil {
			return nil
		}
		f.vars[name] = &variable{value: value, origin: origin}
	case kindAppend:
		if cur == nil {
			f.vars[name] = &variable{value: value, origin: origin}
			return nil
		}
		if cur.simple {
			extra, err := f.expand(value, nil)
			if err != nil {
				return err
			}
			cur.value = strings.TrimSpace(cur.value + " " + extra)
			return nil
		}
		if strings.TrimSpace(cur.value) == "" {
			cur.value = value
		} else {
			cur.value += " " + value
		}
	}
	return nil
}

func kindOf(op string) kind {
	switch op {
	case ":=", "::=":
		return kindSimple
	case "!=":
		return kindShell
	case "?=":
		return kindConditional
	case "+=":
		return kindAppend
	default:
		return kindRecursive
	}
}

func (f *File) lookup(name string) (*variable, bool) {
	v, ok := f.vars[name]
	return v, ok
}

func exists(dir, path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(workflow.Join(dir, path))
	return err == nil
}

func uniq(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func syntax(path string, line int, msg string) error {
	return fmt.Errorf("make: %s:%d: %w: %s", path, line, ErrSyntax, msg)
}
