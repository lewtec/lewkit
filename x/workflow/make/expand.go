package make

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	lewpath "github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/workflow"
)

const maxExpand = 64

func (f *File) expand(s string, auto map[string]string) (string, error) {
	return f.expandDepth(s, auto, 0)
}

func (f *File) expandDepth(s string, auto map[string]string, depth int) (string, error) {
	if depth > maxExpand {
		return "", errors.New("make: variable recursion")
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '$' {
			b.WriteByte(s[i])
			continue
		}
		if i+1 >= len(s) {
			b.WriteByte('$')
			break
		}
		i++
		switch s[i] {
		case '$':
			b.WriteByte('$')
		case '(', '{':
			open := s[i]
			close := byte(')')
			if open == '{' {
				close = '}'
			}
			body, next, err := scanClose(s, i, open, close)
			if err != nil {
				return "", err
			}
			i = next
			part, err := f.expandRef(body, auto, depth)
			if err != nil {
				return "", err
			}
			b.WriteString(part)
		default:
			part, err := f.expandRef(string(s[i]), auto, depth)
			if err != nil {
				return "", err
			}
			b.WriteString(part)
		}
	}
	return b.String(), nil
}

// scanClose returns the text inside one $(...) or ${...}.
// A nested reference is skipped. A bare parenthesis of the same kind
// also nests, which is how $(shell if (true); then echo y; fi) keeps
// the whole script. A one-character reference such as $$ or $< is not
// a nest, so its second byte is not a closer.
func scanClose(s string, openAt int, open, close byte) (string, int, error) {
	depth := 1
	for i := openAt + 1; i < len(s); i++ {
		if s[i] == '$' && i+1 < len(s) {
			switch s[i+1] {
			case '$':
				i++
			case '(', '{':
				closer := byte(')')
				if s[i+1] == '{' {
					closer = '}'
				}
				_, end, err := scanClose(s, i+1, s[i+1], closer)
				if err != nil {
					return "", 0, err
				}
				i = end
			default:
				i++
			}
			continue
		}
		if s[i] == open {
			depth++
			continue
		}
		if s[i] != close {
			continue
		}
		depth--
		if depth == 0 {
			return s[openAt+1 : i], i, nil
		}
	}
	near := s[openAt:]
	if len(near) > 80 {
		near = near[:80]
	}
	return "", 0, fmt.Errorf("make: unterminated variable near %q", near)
}

func (f *File) expandRef(body string, auto map[string]string, depth int) (string, error) {
	if name, rest, ok := cutFunc(body); ok {
		return f.callFunc(name, rest, auto, depth)
	}
	if name, pat, rep, ok := cutSubst(body); ok {
		val, err := f.varValue(name, auto, depth)
		if err != nil {
			return "", err
		}
		pat, err = f.expandDepth(pat, auto, depth+1)
		if err != nil {
			return "", err
		}
		rep, err = f.expandDepth(rep, auto, depth+1)
		if err != nil {
			return "", err
		}
		// $(var:.c=.o) is $(patsubst %.c,%.o,$(var)). A % in the pattern is already patsubst.
		if !strings.Contains(pat, "%") {
			pat = "%" + pat
			rep = "%" + rep
		}
		return patsubst(pat, rep, val), nil
	}
	return f.varValue(strings.TrimSpace(body), auto, depth)
}

func (f *File) varValue(name string, auto map[string]string, depth int) (string, error) {
	if auto != nil {
		if v, ok := auto[name]; ok {
			return v, nil
		}
	}
	v, ok := f.vars[name]
	if !ok {
		return "", nil
	}
	if v.simple {
		return v.value, nil
	}
	return f.expandDepth(v.value, auto, depth+1)
}

func cutFunc(body string) (string, string, bool) {
	s := strings.TrimLeft(body, " \t")
	i := 0
	for i < len(s) && nameChar(s[i]) {
		i++
	}
	if i == 0 || !isFunc(s[:i]) {
		return "", "", false
	}
	rest := s[i:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return "", "", false
	}
	return s[:i], strings.TrimLeft(rest, " \t"), true
}

func isFunc(name string) bool {
	switch name {
	case "wildcard", "patsubst", "subst", "strip", "findstring",
		"filter", "filter-out", "sort", "word", "words", "wordlist",
		"firstword", "lastword", "dir", "notdir", "suffix", "basename",
		"addprefix", "addsuffix", "join", "abspath", "realpath",
		"foreach", "if", "or", "and", "call", "error", "warning",
		"value", "origin", "info", "eval", "shell":
		return true
	default:
		return false
	}
}

func cutSubst(body string) (string, string, string, bool) {
	depth := 0
	colon := -1
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '$':
			if i+1 < len(body) {
				i++
			}
		case '(', '{':
			depth++
		case ')', '}':
			if depth > 0 {
				depth--
			}
		case ':':
			if depth == 0 && colon < 0 {
				colon = i
			}
		case '=':
			if depth == 0 && colon > 0 {
				return strings.TrimSpace(body[:colon]), body[colon+1 : i], body[i+1:], true
			}
		}
	}
	return "", "", "", false
}

func (f *File) callFunc(name, rest string, auto map[string]string, depth int) (string, error) {
	switch name {
	case "foreach":
		return f.fnForeach(rest, auto, depth)
	case "if":
		return f.fnIf(rest, auto, depth)
	case "or":
		return f.fnOr(rest, auto, depth)
	case "and":
		return f.fnAnd(rest, auto, depth)
	case "call":
		return f.fnCall(rest, auto, depth)
	case "origin":
		arg, err := f.expandDepth(rest, auto, depth+1)
		if err != nil {
			return "", err
		}
		return f.fnOrigin(strings.TrimSpace(arg), auto), nil
	case "value":
		arg, err := f.expandDepth(rest, auto, depth+1)
		if err != nil {
			return "", err
		}
		if v, ok := f.vars[strings.TrimSpace(arg)]; ok {
			return v.value, nil
		}
		return "", nil
	case "error":
		arg, err := f.expandDepth(rest, auto, depth+1)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("make: %s", strings.TrimSpace(arg))
	case "warning", "info":
		_, err := f.expandDepth(rest, auto, depth+1)
		return "", err
	case "eval":
		return "", errors.New("make: eval is not supported")
	case "shell":
		arg, err := f.expandDepth(rest, auto, depth+1)
		if err != nil {
			return "", err
		}
		return f.shell(arg)
	default:
		arg, err := f.expandDepth(rest, auto, depth+1)
		if err != nil {
			return "", err
		}
		return f.fnText(name, arg)
	}
}

func (f *File) fnText(name, arg string) (string, error) {
	switch name {
	case "wildcard":
		return f.fnWildcard(arg)
	case "strip":
		return strings.Join(strings.Fields(arg), " "), nil
	case "subst":
		parts := splitComma(arg)
		if len(parts) < 3 {
			return "", nil
		}
		return strings.ReplaceAll(parts[2], parts[0], parts[1]), nil
	case "patsubst":
		parts := splitComma(arg)
		if len(parts) < 3 {
			return "", nil
		}
		return patsubst(parts[0], parts[1], parts[2]), nil
	case "findstring":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		if strings.Contains(parts[1], parts[0]) {
			return parts[0], nil
		}
		return "", nil
	case "filter":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return filterWords(parts[0], parts[1], false), nil
	case "filter-out":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return filterWords(parts[0], parts[1], true), nil
	case "sort":
		return sortWords(arg), nil
	case "word":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return wordAt(parts[0], parts[1]), nil
	case "words":
		return strconv.Itoa(len(strings.Fields(arg))), nil
	case "wordlist":
		parts := splitComma(arg)
		if len(parts) < 3 {
			return "", nil
		}
		return wordList(parts[0], parts[1], parts[2]), nil
	case "firstword":
		fields := strings.Fields(arg)
		if len(fields) == 0 {
			return "", nil
		}
		return fields[0], nil
	case "lastword":
		fields := strings.Fields(arg)
		if len(fields) == 0 {
			return "", nil
		}
		return fields[len(fields)-1], nil
	case "dir":
		return mapWords(arg, dirOf), nil
	case "notdir":
		return mapWords(arg, notDir), nil
	case "suffix":
		return mapWords(arg, suffixOf), nil
	case "basename":
		return mapWords(arg, baseOf), nil
	case "addprefix":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return mapWords(parts[1], func(w string) string { return parts[0] + w }), nil
	case "addsuffix":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return mapWords(parts[1], func(w string) string { return w + parts[0] }), nil
	case "abspath":
		return mapWords(arg, f.absPath), nil
	case "realpath":
		return f.fnRealpath(arg), nil
	case "join":
		parts := splitComma(arg)
		if len(parts) < 2 {
			return "", nil
		}
		return joinWords(parts[0], parts[1]), nil
	default:
		return "", nil
	}
}

func (f *File) absPath(name string) string {
	p := lewpath.New(name)
	if p.IsAbs() {
		return p.String()
	}
	return workflow.Join(f.dir, name)
}

func (f *File) fnWildcard(arg string) (string, error) {
	var out []string
	for _, pattern := range strings.Fields(arg) {
		// * does not cross a slash. A leading ./ is part of the match:
		// $(wildcard ./crt/*.c) is ./crt/crt1.c, which later patsubst uses.
		search := pattern
		if !filepath.IsAbs(pattern) {
			search = filepath.Join(f.dir, filepath.FromSlash(pattern))
		}
		matches, err := filepath.Glob(search)
		if err != nil {
			return "", err
		}
		sort.Strings(matches)
		dot := strings.HasPrefix(pattern, "./")
		for _, match := range matches {
			name := match
			if !filepath.IsAbs(pattern) {
				rel, err := filepath.Rel(f.dir, match)
				if err == nil && !strings.HasPrefix(rel, "..") {
					name = rel
					if dot {
						name = "./" + name
					}
				}
			}
			out = append(out, filepath.ToSlash(name))
		}
	}
	return strings.Join(out, " "), nil
}

func (f *File) fnRealpath(arg string) string {
	var out []string
	for _, word := range strings.Fields(arg) {
		resolved, err := filepath.EvalSymlinks(workflow.Join(f.dir, word))
		if err != nil {
			continue
		}
		out = append(out, filepath.Clean(resolved))
	}
	return strings.Join(out, " ")
}

func (f *File) fnForeach(rest string, auto map[string]string, depth int) (string, error) {
	parts := splitComma(rest)
	if len(parts) < 3 {
		return "", nil
	}
	name, err := f.expandDepth(parts[0], auto, depth+1)
	if err != nil {
		return "", err
	}
	list, err := f.expandDepth(parts[1], auto, depth+1)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	first := true
	for _, word := range strings.Fields(list) {
		var part string
		err := f.withVar(strings.TrimSpace(name), word, func() error {
			var err error
			part, err = f.expandDepth(parts[2], auto, depth+1)
			return err
		})
		if err != nil {
			return "", err
		}
		if !first {
			b.WriteByte(' ')
		}
		first = false
		b.WriteString(part)
	}
	return b.String(), nil
}

func (f *File) fnIf(rest string, auto map[string]string, depth int) (string, error) {
	parts := splitComma(rest)
	if len(parts) == 0 {
		return "", nil
	}
	cond, err := f.expandDepth(parts[0], auto, depth+1)
	if err != nil {
		return "", err
	}
	take := 1
	if strings.TrimSpace(cond) == "" {
		take = 2
	}
	if take >= len(parts) {
		return "", nil
	}
	return f.expandDepth(parts[take], auto, depth+1)
}

func (f *File) fnOr(rest string, auto map[string]string, depth int) (string, error) {
	for _, part := range splitComma(rest) {
		v, err := f.expandDepth(part, auto, depth+1)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(v) != "" {
			return v, nil
		}
	}
	return "", nil
}

func (f *File) fnAnd(rest string, auto map[string]string, depth int) (string, error) {
	parts := splitComma(rest)
	var last string
	for _, part := range parts {
		v, err := f.expandDepth(part, auto, depth+1)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(v) == "" {
			return "", nil
		}
		last = v
	}
	return last, nil
}

func (f *File) fnCall(rest string, auto map[string]string, depth int) (string, error) {
	parts := splitComma(rest)
	if len(parts) == 0 {
		return "", nil
	}
	name, err := f.expandDepth(strings.TrimSpace(parts[0]), auto, depth+1)
	if err != nil {
		return "", err
	}
	args := make([]string, 0, len(parts)-1)
	for _, part := range parts[1:] {
		v, err := f.expandDepth(part, auto, depth+1)
		if err != nil {
			return "", err
		}
		args = append(args, v)
	}
	v, ok := f.vars[strings.TrimSpace(name)]
	if !ok {
		return "", nil
	}
	var out string
	err = f.withCall(strings.TrimSpace(name), args, func() error {
		if v.simple {
			out = v.value
			return nil
		}
		out, err = f.expandDepth(v.value, auto, depth+1)
		return err
	})
	return out, err
}

func (f *File) fnOrigin(name string, auto map[string]string) string {
	if auto != nil {
		if _, ok := auto[name]; ok {
			return "automatic"
		}
	}
	v, ok := f.vars[name]
	if !ok {
		return "undefined"
	}
	switch v.origin {
	case originDefault:
		return "default"
	case originEnv:
		return "environment"
	case originFile:
		return "file"
	case originCommand:
		return "command line"
	case originOverride:
		return "override"
	case originAutomatic:
		return "automatic"
	default:
		return "undefined"
	}
}

func (f *File) withVar(name, value string, fn func() error) error {
	prev, had := f.vars[name]
	f.vars[name] = &variable{value: value, simple: true, origin: originAutomatic}
	err := fn()
	if had {
		f.vars[name] = prev
	} else {
		delete(f.vars, name)
	}
	return err
}

func (f *File) withCall(name string, args []string, fn func() error) error {
	type saved struct {
		name string
		v    *variable
		had  bool
	}
	var old []saved
	set := func(n, value string) {
		prev, had := f.vars[n]
		old = append(old, saved{n, prev, had})
		f.vars[n] = &variable{value: value, simple: true, origin: originAutomatic}
	}
	set("0", name)
	for i, arg := range args {
		set(strconv.Itoa(i+1), arg)
	}
	err := fn()
	for i := len(old) - 1; i >= 0; i-- {
		if old[i].had {
			f.vars[old[i].name] = old[i].v
		} else {
			delete(f.vars, old[i].name)
		}
	}
	return err
}

// shell runs script. A non-zero exit still returns the output, matching
// GNU make. A cancelled context is returned so the load stops with it.
func (f *File) shell(script string) (string, error) {
	if f.ctx == nil {
		return "", errors.New("make: nil context")
	}
	if err := f.ctx.Err(); err != nil {
		return "", err
	}
	cmd := execdriver.MustCommand("sh", "-c", script)
	cmd.Dir = f.dir
	out, _ := execdriver.Output(f.ctx, cmd)
	if err := f.ctx.Err(); err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(out), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", " ")
	return strings.TrimRight(text, " "), nil
}

func splitComma(s string) []string {
	var args []string
	var b strings.Builder
	depth := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '$' && i+1 < len(s) {
			next := s[i+1]
			b.WriteByte(c)
			b.WriteByte(next)
			i++
			// $( and ${ nest. $$ is a literal dollar, so its following
			// byte is parsed on the next iteration.
			if next == '(' || next == '{' {
				depth++
			}
			continue
		}
		if c == '(' || c == '{' {
			depth++
		} else if (c == ')' || c == '}') && depth > 0 {
			depth--
		} else if c == ',' && depth == 0 {
			args = append(args, b.String())
			b.Reset()
			continue
		}
		b.WriteByte(c)
	}
	args = append(args, b.String())
	return args
}

func patsubst(pattern, repl, text string) string {
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = patsubstWord(pattern, repl, word)
	}
	return strings.Join(words, " ")
}

func patsubstWord(pattern, repl, word string) string {
	stem, ok := stemOf(pattern, word)
	if !ok {
		if !strings.Contains(pattern, "%") && word == pattern {
			return repl
		}
		return word
	}
	ri := strings.IndexByte(repl, '%')
	if ri < 0 {
		return repl
	}
	return repl[:ri] + stem + repl[ri+1:]
}

func stemOf(pattern, name string) (string, bool) {
	i := strings.IndexByte(pattern, '%')
	if i < 0 {
		if pattern == name {
			return "", true
		}
		return "", false
	}
	pre, suf := pattern[:i], pattern[i+1:]
	if len(name) < len(pre)+len(suf) {
		return "", false
	}
	if !strings.HasPrefix(name, pre) || !strings.HasSuffix(name, suf) {
		return "", false
	}
	return name[len(pre) : len(name)-len(suf)], true
}

func applyStem(words []string, stem string) []string {
	out := make([]string, len(words))
	for i, word := range words {
		out[i] = strings.ReplaceAll(word, "%", stem)
	}
	return out
}

func filterWords(patterns, text string, drop bool) string {
	pats := strings.Fields(patterns)
	var keep []string
	for _, word := range strings.Fields(text) {
		match := false
		for _, pattern := range pats {
			if _, ok := stemOf(pattern, word); ok {
				match = true
				break
			}
			if !strings.Contains(pattern, "%") && word == pattern {
				match = true
				break
			}
		}
		if match == !drop {
			keep = append(keep, word)
		}
	}
	return strings.Join(keep, " ")
}

func sortWords(text string) string {
	words := strings.Fields(text)
	sort.Strings(words)
	out := words[:0]
	var prev string
	for i, word := range words {
		if i > 0 && word == prev {
			continue
		}
		prev = word
		out = append(out, word)
	}
	return strings.Join(out, " ")
}

func wordAt(index, text string) string {
	n, err := strconv.Atoi(strings.TrimSpace(index))
	fields := strings.Fields(text)
	if err != nil || n < 1 || n > len(fields) {
		return ""
	}
	return fields[n-1]
}

func wordList(start, end, text string) string {
	s, err1 := strconv.Atoi(strings.TrimSpace(start))
	e, err2 := strconv.Atoi(strings.TrimSpace(end))
	fields := strings.Fields(text)
	if err1 != nil || err2 != nil || len(fields) == 0 {
		return ""
	}
	if s < 1 {
		s = 1
	}
	if e > len(fields) {
		e = len(fields)
	}
	if s > e {
		return ""
	}
	return strings.Join(fields[s-1:e], " ")
}

func mapWords(text string, fn func(string) string) string {
	words := strings.Fields(text)
	for i, word := range words {
		words[i] = fn(word)
	}
	return strings.Join(words, " ")
}

// dirOf is make's $(dir). A trailing slash names that directory:
// "a/b/" stays "a/b", the word before the slash.
func dirOf(name string) string {
	if name == "" {
		return "."
	}
	if strings.HasSuffix(name, "/") {
		trimmed := strings.TrimSuffix(name, "/")
		if trimmed == "" {
			return "/"
		}
		return trimmed
	}
	return lewpath.New(name).Parent().String()
}

func notDir(name string) string {
	if name == "" || strings.HasSuffix(name, "/") {
		return ""
	}
	return lewpath.New(name).Name()
}

func suffixOf(name string) string {
	base := notDir(name)
	if base == "" {
		return ""
	}
	return lewpath.New(base).Suffix()
}

func baseOf(name string) string {
	suf := suffixOf(name)
	if suf == "" {
		return name
	}
	return strings.TrimSuffix(name, suf)
}

func joinWords(a, b string) string {
	left := strings.Fields(a)
	right := strings.Fields(b)
	n := len(left)
	if len(right) > n {
		n = len(right)
	}
	out := make([]string, 0, n)
	for i := range n {
		var word string
		if i < len(left) {
			word += left[i]
		}
		if i < len(right) {
			word += right[i]
		}
		out = append(out, word)
	}
	return strings.Join(out, " ")
}
