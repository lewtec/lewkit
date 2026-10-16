// Package ignore decorates an [io/fs.FS] with gitignore rules.
//
// [Collect] reads rules from the wrapped filesystem in this order:
// optional skip-directory names, .git/info/exclude, parent .gitignore
// files, nested .gitignore files, then optional .gitattributes marks.
// Later rules win. A parent-directory rule such as build/ is not undone
// by a later negation of a file inside it: an excluded directory stays
// excluded.
//
// The returned [FS] hides ignored names. A directory stays visible when
// a later negation could still match something inside it.
package ignore

import (
	"bytes"
	"io/fs"
	"strings"

	"github.com/git-pkgs/gitignore"
	"github.com/lewtec/lewkit/x/errs"
	lewfs "github.com/lewtec/lewkit/x/fs"
	lewpath "github.com/lewtec/lewkit/x/path"
)

// errNilFS is [Collect] on a nil filesystem.
var errNilFS = errs.Wrap(errs.ErrMissing, "nil fs")

// Config selects rules that are not read from gitignore files.
type Config struct {
	// SkipDirs are directory basenames ignored before any git file.
	// A name matches that directory at any depth in the wrapped filesystem.
	SkipDirs []string

	// Attributes reads linguist-generated and refactree-ignored marks
	// from .gitattributes. Unset marks are negations.
	Attributes bool
}

// FS is fsys with gitignore rules applied.
// Names passed to [FS.Ignored] and [FS.SkipDir] are slash-separated paths
// relative to the wrapped root, the same form [io/fs] uses.
type FS struct {
	base fs.FS

	matcher   *gitignore.Matcher
	negations []negation
}

var (
	_ fs.FS          = (*FS)(nil)
	_ fs.ReadDirFS   = (*FS)(nil)
	_ fs.StatFS      = (*FS)(nil)
	_ fs.ReadDirFile = (*dirFile)(nil)
)

// Ignored reports whether name is ignored by the collected rules.
func (f *FS) Ignored(name string) bool {
	if f == nil || f.matcher == nil || name == "." || name == "" {
		return false
	}
	return f.ignored(name, isDirectory(f.base, name))
}

// SkipDir reports whether a walk should not enter the directory name.
// It returns false when name is not ignored, and when a later negation
// could still match something inside it.
func (f *FS) SkipDir(name string) bool {
	if f == nil || f.matcher == nil || name == "." || name == "" {
		return false
	}
	if !f.ignored(name, true) {
		return false
	}
	return !f.negationCouldApply(name)
}

// Open opens name in the wrapped filesystem.
// An ignored file, and a directory that [FS.SkipDir] refuses, is [fs.ErrNotExist].
func (f *FS) Open(name string) (fs.File, error) {
	if f == nil {
		return nil, errNilFS
	}
	name, err := normalize("open", name)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(f.base, name)
	if err != nil {
		return nil, err
	}
	if name != "." && f.concealed(name, info.IsDir()) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	file, err := f.base.Open(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return file, nil
	}
	return &dirFile{File: file, owner: f, name: name}, nil
}

// Stat stats name in the wrapped filesystem.
// An ignored file, and a directory that [FS.SkipDir] refuses, is [fs.ErrNotExist].
func (f *FS) Stat(name string) (fs.FileInfo, error) {
	if f == nil {
		return nil, errNilFS
	}
	name, err := normalize("stat", name)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(f.base, name)
	if err != nil {
		return nil, err
	}
	if name != "." && f.concealed(name, info.IsDir()) {
		return nil, &fs.PathError{Op: "stat", Path: name, Err: fs.ErrNotExist}
	}
	return info, nil
}

// ReadDir lists name, omitting ignored files and directories [FS.SkipDir] refuses.
func (f *FS) ReadDir(name string) ([]fs.DirEntry, error) {
	if f == nil {
		return nil, errNilFS
	}
	name, err := normalize("readdir", name)
	if err != nil {
		return nil, err
	}
	info, err := fs.Stat(f.base, name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrInvalid}
	}
	if name != "." && f.SkipDir(name) {
		return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
	}
	entries, err := fs.ReadDir(f.base, name)
	if err != nil {
		return nil, err
	}
	return f.filter(name, entries), nil
}

// concealed reports whether Open, Stat, and ReadDir should hide name.
// Directories use [FS.SkipDir] so a later negation can still be reached.
func (f *FS) concealed(name string, directory bool) bool {
	if directory {
		return f.SkipDir(name)
	}
	return f.ignored(name, false)
}

func (f *FS) filter(dir string, entries []fs.DirEntry) []fs.DirEntry {
	visible := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if name == "" || name == "." || name == ".." {
			continue
		}
		full := name
		if dir != "." && dir != "" {
			full = dir + "/" + name
		}
		if f.concealed(full, entry.IsDir()) {
			continue
		}
		visible = append(visible, entry)
	}
	return visible
}

type dirFile struct {
	fs.File
	owner   *FS
	name    string
	entries []fs.DirEntry
	off     int
	loaded  bool
}

func (d *dirFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !d.loaded {
		directory, ok := d.File.(fs.ReadDirFile)
		if !ok {
			return nil, &fs.PathError{Op: "readdir", Path: d.name, Err: fs.ErrInvalid}
		}
		entries, err := directory.ReadDir(-1)
		if err != nil {
			return nil, err
		}
		d.entries = d.owner.filter(d.name, entries)
		d.loaded = true
	}
	return lewfs.DirEntries(d.entries, &d.off, n)
}

func (f *FS) ignored(name string, directory bool) bool {
	if f.parentExcluded(name) {
		return true
	}
	return f.matcher.MatchPath(name, directory)
}

// parentExcluded reports whether a path sits inside a directory that git
// will not list. A negation of a file under that directory has no effect.
// A trailing /** does not exclude the directory itself, so a negation
// inside it can still match.
func (f *FS) parentExcluded(name string) bool {
	parts := strings.Split(name, "/")
	for i := 1; i < len(parts); i++ {
		if f.directoryExcluded(strings.Join(parts[:i], "/")) {
			return true
		}
	}
	return false
}

func (f *FS) directoryExcluded(name string) bool {
	detail := f.matcher.MatchDetail(name + "/")
	if !detail.Ignored {
		return false
	}
	return !strings.HasSuffix(detail.Pattern, "/**")
}

func (f *FS) negationCouldApply(directory string) bool {
	detail := f.matcher.MatchDetail(directory + "/")
	if !detail.Ignored || !strings.HasSuffix(detail.Pattern, "/**") {
		return false
	}
	for _, rule := range f.negations {
		if rule.couldMatchUnder(directory) {
			return true
		}
	}
	return false
}

func (f *FS) addPatterns(data []byte, scope string) {
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	if len(data) == 0 {
		return
	}
	f.matcher.AddPatterns(data, scope)
	f.negations = append(f.negations, negationsIn(data, scope)...)
}

func normalize(op, name string) (string, error) {
	if name == "" {
		name = "."
	}
	if err := lewfs.CheckName(op, name); err != nil {
		return "", err
	}
	return name, nil
}

func isDirectory(fsys fs.FS, name string) bool {
	if strings.HasSuffix(name, "/") {
		return true
	}
	info, err := fs.Stat(fsys, name)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func notDirectory(op, name string) error {
	return &fs.PathError{Op: op, Path: name, Err: lewpath.ErrNotDir}
}

// negation is a ! rule, scoped to the directory of the file that declared it.
type negation struct {
	scope   string
	pattern string
}

func (n negation) couldMatchUnder(directory string) bool {
	pattern := n.pattern
	if pattern == "" {
		return false
	}
	if !strings.Contains(pattern, "/") {
		return withinScope(n.scope, directory)
	}
	full := joinSlash(n.scope, strings.Trim(pattern, "/"))
	if strings.ContainsAny(pattern, "*?[") {
		return withinScope(n.scope, directory) ||
			strings.HasPrefix(full, directory+"/") ||
			strings.HasPrefix(directory, full+"/")
	}
	return full == directory || strings.HasPrefix(full, directory+"/")
}

func withinScope(scope, directory string) bool {
	if scope == "" {
		return true
	}
	return directory == scope || strings.HasPrefix(directory, scope+"/")
}

func joinSlash(scope, pattern string) string {
	if scope == "" {
		return pattern
	}
	if pattern == "" {
		return scope
	}
	return scope + "/" + pattern
}

func negationsIn(data []byte, scope string) []negation {
	var rules []negation
	for line := range strings.SplitSeq(string(data), "\n") {
		line = trimGitignoreLine(line)
		if line == "" || line[0] == '#' || strings.HasPrefix(line, `\`) {
			continue
		}
		if !strings.HasPrefix(line, "!") {
			continue
		}
		pattern := line[1:]
		if pattern == "" {
			continue
		}
		rules = append(rules, negation{scope: scope, pattern: pattern})
	}
	return rules
}

func trimGitignoreLine(line string) string {
	line = strings.TrimRight(line, "\r")
	for len(line) > 0 && line[len(line)-1] == ' ' {
		if len(line) >= 2 && line[len(line)-2] == '\\' {
			break
		}
		line = line[:len(line)-1]
	}
	return line
}
