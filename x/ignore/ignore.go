// Package ignore collects gitignore rules for one tree.
//
// [Collect] reads rules in this order: optional skip-directory names,
// .git/info/exclude, parent .gitignore files, nested .gitignore files,
// then optional .gitattributes marks. Later rules win. A parent-directory
// rule such as build/ is not undone by a later negation of a file inside
// it, matching gitignore: an excluded directory stays excluded.
package ignore

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/git-pkgs/gitignore"
)

// Config selects rules that are not read from gitignore files.
type Config struct {
	// SkipDirs are directory basenames ignored before any git file.
	// A name matches that directory at any depth under the collected root.
	SkipDirs []string

	// Attributes reads linguist-generated and refactree-ignored marks
	// from .gitattributes. Unset marks are negations.
	Attributes bool
}

// Set is the rules collected for one tree.
type Set struct {
	// Root is the absolute directory paths are matched against.
	// It is the git toplevel when the collected directory is inside a
	// checkout, and the collected directory otherwise.
	Root string

	matcher   *gitignore.Matcher
	negations []negation
}

// Ignored reports whether path is ignored by the collected rules.
func (s *Set) Ignored(path string) bool {
	if s == nil || s.matcher == nil {
		return false
	}
	relative, ok := s.relative(path)
	if !ok || relative == "." {
		return false
	}
	return s.ignored(relative, isDirectory(path))
}

// SkipDir reports whether a walk should not enter the directory path.
// It returns false when path is not ignored, and when a later negation
// could still match something inside it.
func (s *Set) SkipDir(path string) bool {
	if s == nil || s.matcher == nil {
		return false
	}
	relative, ok := s.relative(path)
	if !ok || relative == "." {
		return false
	}
	if !s.ignored(relative, true) {
		return false
	}
	return !s.negationCouldApply(relative)
}

func (s *Set) ignored(relative string, directory bool) bool {
	if s.parentExcluded(relative) {
		return true
	}
	return s.matcher.MatchPath(relative, directory)
}

// parentExcluded reports whether a path sits inside a directory that git
// will not list. A negation of a file under that directory has no effect.
// A trailing /** does not exclude the directory itself, so a negation
// inside it can still match.
func (s *Set) parentExcluded(relative string) bool {
	parts := strings.Split(relative, "/")
	for i := 1; i < len(parts); i++ {
		if s.directoryExcluded(strings.Join(parts[:i], "/")) {
			return true
		}
	}
	return false
}

func (s *Set) directoryExcluded(relative string) bool {
	detail := s.matcher.MatchDetail(relative + "/")
	if !detail.Ignored {
		return false
	}
	return !strings.HasSuffix(detail.Pattern, "/**")
}

func (s *Set) negationCouldApply(directory string) bool {
	detail := s.matcher.MatchDetail(directory + "/")
	if !detail.Ignored || !strings.HasSuffix(detail.Pattern, "/**") {
		return false
	}
	for _, rule := range s.negations {
		if rule.couldMatchUnder(directory) {
			return true
		}
	}
	return false
}

func (s *Set) relative(path string) (string, bool) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	absolute = filepath.Clean(absolute)
	relative, err := filepath.Rel(s.Root, absolute)
	if err != nil {
		return "", false
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relative), true
}

func (s *Set) addPatterns(data []byte, scope string) {
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	if len(data) == 0 {
		return
	}
	s.matcher.AddPatterns(data, scope)
	s.negations = append(s.negations, negationsIn(data, scope)...)
}

func isDirectory(path string) bool {
	if strings.HasSuffix(path, "/") || strings.HasSuffix(path, string(os.PathSeparator)) {
		return true
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
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
		return withinScope(n.scope, directory) || strings.HasPrefix(full, directory+"/") || strings.HasPrefix(directory, full+"/")
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
