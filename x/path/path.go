package path

import (
	"errors"
	stdpath "path"
	"slices"
	"strconv"
	"strings"

	"io/fs"
)

// Path is an immutable [io/fs] name. It does no I/O.
type Path struct {
	s string
}

// New joins segments with [path.Join]. Empty New is ".".
// A later absolute segment drops the ones before it.
func New(parts ...string) Path {
	if len(parts) == 0 {
		return Path{s: "."}
	}
	start := 0
	for i, part := range parts {
		if stdpath.IsAbs(part) {
			start = i
		}
	}
	s := stdpath.Join(parts[start:]...)
	if s == "" {
		return Path{s: "."}
	}
	return Path{s: s}
}

// String returns the slash-separated name.
func (p Path) String() string { return p.s }

// Join appends segments to p.
func (p Path) Join(parts ...string) Path {
	if len(parts) == 0 {
		return p
	}
	return New(append([]string{p.s}, parts...)...)
}

// Parent is the directory of p. Parent of "." or "/" is itself.
func (p Path) Parent() Path {
	return Path{s: stdpath.Dir(p.s)}
}

// Name is the last component of p.
func (p Path) Name() string {
	return stdpath.Base(p.s)
}

// Suffix is the last extension of [Path.Name], including the dot.
// Name ".bashrc" has an empty suffix.
func (p Path) Suffix() string {
	name := p.Name()
	ext := stdpath.Ext(name)
	if ext == "" || ext == name {
		return ""
	}
	return ext
}

// Stem is [Path.Name] without [Path.Suffix].
func (p Path) Stem() string {
	name := p.Name()
	if suf := p.Suffix(); suf != "" {
		name = strings.TrimSuffix(name, suf)
	}
	return name
}

// Suffixes is every extension of [Path.Name], last one last.
func (p Path) Suffixes() []string {
	name := p.Name()
	var out []string
	for {
		ext := stdpath.Ext(name)
		if ext == "" || ext == name {
			break
		}
		out = append(out, ext)
		name = strings.TrimSuffix(name, ext)
	}
	slices.Reverse(out)
	return out
}

// Parts is the components of p. The root name is ["."].
func (p Path) Parts() []string {
	s := p.s
	if s == "." || s == "" {
		return []string{"."}
	}
	if s == "/" {
		return []string{"/"}
	}
	var parts []string
	if stdpath.IsAbs(s) {
		parts = append(parts, "/")
		s = strings.TrimPrefix(s, "/")
	}
	if s != "" {
		parts = append(parts, strings.Split(s, "/")...)
	}
	return parts
}

// IsAbs reports a leading slash. Absolute names are not valid [io/fs] names.
func (p Path) IsAbs() bool {
	return stdpath.IsAbs(p.s)
}

// Valid reports whether p is a legal [io/fs] name. "." is valid.
func (p Path) Valid() bool {
	return p.s == "." || fs.ValidPath(p.s)
}

// Clean applies [path.Clean].
func (p Path) Clean() Path {
	return Path{s: stdpath.Clean(p.s)}
}

// WithName replaces the last component.
func (p Path) WithName(name string) Path {
	return p.Parent().Join(name)
}

// WithStem replaces the stem and keeps [Path.Suffix].
func (p Path) WithStem(stem string) Path {
	return p.Parent().Join(stem + p.Suffix())
}

// WithSuffix replaces the last suffix. suffix is empty or starts with a dot.
func (p Path) WithSuffix(suffix string) Path {
	return p.Parent().Join(p.Stem() + suffix)
}

// Match reports whether p matches pattern, using [path.Match].
func (p Path) Match(pattern string) (bool, error) {
	return stdpath.Match(pattern, p.s)
}

// Rel returns p relative to base. It walks up, like [path/filepath.Rel].
func (p Path) Rel(base Path) (Path, error) {
	s, err := rel(base.s, p.s)
	if err != nil {
		return Path{}, err
	}
	return Path{s: s}, nil
}

func rel(base, target string) (string, error) {
	base = stdpath.Clean(base)
	target = stdpath.Clean(target)
	if stdpath.IsAbs(base) != stdpath.IsAbs(target) {
		return "", &RelError{Target: target, Base: base}
	}
	if base == target {
		return ".", nil
	}
	bp := relParts(base)
	tp := relParts(target)
	i := 0
	for i < len(bp) && i < len(tp) && bp[i] == tp[i] {
		i++
	}
	var out []string
	for j := i; j < len(bp); j++ {
		if bp[j] == ".." {
			return "", &RelError{Target: target, Base: base}
		}
		out = append(out, "..")
	}
	out = append(out, tp[i:]...)
	if len(out) == 0 {
		return ".", nil
	}
	return strings.Join(out, "/"), nil
}

func relParts(s string) []string {
	if s == "." {
		return nil
	}
	if s == "/" {
		return []string{""}
	}
	return strings.Split(s, "/")
}

var (
	// ErrEmptyPath is an empty OS path passed to [Open].
	ErrEmptyPath = errors.New("empty path")
	// ErrNotDir is [Open] on a path that exists and is not a directory.
	ErrNotDir = errors.New("not a directory")
	// ErrReadOnly is a write method on a filesystem that lacks that method.
	ErrReadOnly = errors.New("read-only filesystem")
	// ErrRel is [Path.Rel] when the names cannot be made relative.
	ErrRel = errors.New("cannot be relative")
)

// RelError is a [Path.Rel] that cannot be computed.
type RelError struct {
	// Target is the name being made relative.
	Target string
	// Base is the name Target is relative to.
	Base string
}

// Error describes the failed [Path.Rel].
func (e *RelError) Error() string {
	return "rel " + strconv.Quote(e.Target) + ": cannot be relative to " + strconv.Quote(e.Base)
}

// Unwrap returns [ErrRel].
func (e *RelError) Unwrap() error { return ErrRel }
