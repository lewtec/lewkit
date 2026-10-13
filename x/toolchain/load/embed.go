package load

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/mod/module"
)

// resolveEmbed matches //go:embed patterns the way the go command does.
// all: includes names that start with "." or "_". Symlinks are rejected.
func resolveEmbed(pkgdir string, patterns []string) (*Embed, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	pmap := map[string][]string{}
	have := map[string]int{}
	dirOK := map[string]bool{}
	for pid, pattern := range patterns {
		pid++
		glob, all := strings.CutPrefix(pattern, "all:")
		if _, err := path.Match(glob, ""); err != nil || !validEmbedPattern(glob) {
			return nil, fmt.Errorf("%w: %s: invalid pattern syntax", ErrEmbed, pattern)
		}
		matches, err := filepath.Glob(quoteGlob(pkgdir) + string(filepath.Separator) + filepath.FromSlash(glob))
		if err != nil {
			return nil, fmt.Errorf("%w: %s: %v", ErrEmbed, pattern, err)
		}
		var list []string
		for _, file := range matches {
			rel := filepath.ToSlash(trimPrefix(file, pkgdir))
			info, err := os.Lstat(file)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %v", ErrEmbed, pattern, err)
			}
			what := "file"
			if info.IsDir() {
				what = "directory"
			}
			for dir := file; len(dir) > len(pkgdir)+1 && !dirOK[dir]; dir = filepath.Dir(dir) {
				if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
					return nil, fmt.Errorf("%w: %s: cannot embed %s %s: in different module", ErrEmbed, pattern, what, rel)
				}
				if dir != file {
					if st, err := os.Lstat(dir); err == nil && !st.IsDir() {
						return nil, fmt.Errorf("%w: %s: cannot embed %s %s: in non-directory %s", ErrEmbed, pattern, what, rel, dir[len(pkgdir)+1:])
					}
				}
				dirOK[dir] = true
				elem := filepath.Base(dir)
				if isBadEmbedName(elem) {
					if dir == file {
						return nil, fmt.Errorf("%w: %s: cannot embed %s %s: invalid name %s", ErrEmbed, pattern, what, rel, elem)
					}
					return nil, fmt.Errorf("%w: %s: cannot embed %s %s: in invalid directory %s", ErrEmbed, pattern, what, rel, elem)
				}
			}
			switch {
			case info.Mode().IsRegular():
				if have[rel] != pid {
					have[rel] = pid
					list = append(list, rel)
				}
			case info.IsDir():
				count := 0
				err := filepath.WalkDir(file, func(walkPath string, d fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					walkRel := filepath.ToSlash(trimPrefix(walkPath, pkgdir))
					name := d.Name()
					if walkPath != file && (isBadEmbedName(name) || ((name[0] == '.' || name[0] == '_') && !all)) {
						if d.IsDir() {
							return fs.SkipDir
						}
						if name[0] == '.' || name[0] == '_' {
							return nil
						}
						return fmt.Errorf("cannot embed file %s: invalid name %s", walkRel, name)
					}
					if d.IsDir() {
						if _, err := os.Stat(filepath.Join(walkPath, "go.mod")); err == nil && walkPath != file {
							return fs.SkipDir
						}
						return nil
					}
					if !d.Type().IsRegular() {
						return nil
					}
					count++
					if have[walkRel] != pid {
						have[walkRel] = pid
						list = append(list, walkRel)
					}
					return nil
				})
				if err != nil {
					return nil, fmt.Errorf("%w: %s: %v", ErrEmbed, pattern, err)
				}
				if count == 0 {
					return nil, fmt.Errorf("%w: %s: cannot embed directory %s: contains no embeddable files", ErrEmbed, pattern, rel)
				}
			default:
				return nil, fmt.Errorf("%w: %s: cannot embed irregular file %s", ErrEmbed, pattern, rel)
			}
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("%w: %s: no matching files found", ErrEmbed, pattern)
		}
		sort.Strings(list)
		pmap[pattern] = list
	}
	emb := &Embed{Patterns: pmap, Files: map[string]string{}}
	for _, list := range pmap {
		for _, file := range list {
			emb.Files[file] = filepath.Join(pkgdir, filepath.FromSlash(file))
		}
	}
	return emb, nil
}

func validEmbedPattern(pattern string) bool {
	return pattern != "." && fs.ValidPath(pattern)
}

func isBadEmbedName(name string) bool {
	if err := module.CheckFilePath(name); err != nil {
		return true
	}
	switch name {
	case "", ".bzr", ".hg", ".git", ".svn":
		return true
	}
	return false
}

func trimPrefix(path, dir string) string {
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return strings.TrimPrefix(path, dir+string(filepath.Separator))
	}
	return rel
}

func quoteGlob(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '*', '?', '[', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}
