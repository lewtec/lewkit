package term

import (
	"fmt"
	"sort"
	"strings"
)

// cleanPath returns a slash-separated key with no leading or trailing slash.
// An empty string is the root. "." is the root. ".." and blank segments are rejected.
func cleanPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")
	path = strings.Trim(path, "/")
	if path == "" || path == "." {
		return "", nil
	}
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if !validSegment(part) {
			return "", fmt.Errorf("%w: %s", ErrPath, path)
		}
	}
	return strings.Join(parts, "/"), nil
}

func validSegment(seg string) bool {
	if seg == "" || seg == "." || seg == ".." {
		return false
	}
	for _, r := range seg {
		if r <= 0x20 || r == 0x7f || r == '/' || r == '\\' {
			return false
		}
	}
	return true
}

// assignPath chooses the file key for a new pane.
// An empty request stores the file at id. A trailing slash stores it inside
// that folder, named by id. The key is not a host filesystem path.
func assignPath(requested, id string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return id, nil
	}
	folder := strings.HasSuffix(requested, "/")
	cleaned, err := cleanPath(strings.TrimRight(requested, "/"))
	if err != nil {
		return "", err
	}
	if folder {
		if cleaned == "" {
			return id, nil
		}
		return cleaned + "/" + id, nil
	}
	if cleaned == "" {
		return "", fmt.Errorf("%w: %s", ErrPath, requested)
	}
	return cleaned, nil
}

// resolvePath joins text onto current.
// A leading slash starts at the root. ".." climbs one folder.
// Empty text is the root when current is also ignored by a leading slash,
// and empty text alone is the root.
func resolvePath(current, text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", nil
	}
	var parts []string
	if !strings.HasPrefix(text, "/") && current != "" {
		parts = strings.Split(current, "/")
	}
	text = strings.Trim(text, "/")
	if text == "" {
		return strings.Join(parts, "/"), nil
	}
	for _, seg := range strings.Split(text, "/") {
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			if len(parts) == 0 {
				return "", fmt.Errorf("%w: %s", ErrPath, text)
			}
			parts = parts[:len(parts)-1]
			continue
		}
		if !validSegment(seg) {
			return "", fmt.Errorf("%w: %s", ErrPath, seg)
		}
		parts = append(parts, seg)
	}
	return strings.Join(parts, "/"), nil
}

func pathParent(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return ""
	}
	return path[:i]
}

func pathBase(path string) string {
	i := strings.LastIndex(path, "/")
	if i < 0 {
		return path
	}
	return path[i+1:]
}

func pathUnder(path, prefix string) bool {
	if prefix == "" {
		return true
	}
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

func pathTaken(paths map[string]string, path string) bool {
	if path == "" {
		return false
	}
	if _, ok := paths[path]; ok {
		return true
	}
	prefix := path + "/"
	for existing := range paths {
		if strings.HasPrefix(existing, prefix) || strings.HasPrefix(path, existing+"/") {
			return true
		}
	}
	return false
}

func catalog(order []string, panes map[string]*live, stored []string) []Entry {
	dirs := map[string]struct{}{}
	for _, dir := range stored {
		if dir == "" {
			continue
		}
		dirs[dir] = struct{}{}
		for parent := pathParent(dir); parent != ""; parent = pathParent(parent) {
			dirs[parent] = struct{}{}
		}
	}
	files := make([]Entry, 0, len(order))
	for _, id := range order {
		pane := panes[id]
		if pane == nil {
			continue
		}
		path := pane.info.Path
		if path == "" {
			path = id
		}
		files = append(files, Entry{Name: pathBase(path), Path: path, Pane: id})
		for parent := pathParent(path); parent != ""; parent = pathParent(parent) {
			dirs[parent] = struct{}{}
		}
	}
	entries := make([]Entry, 0, len(dirs)+len(files))
	dirPaths := make([]string, 0, len(dirs))
	for dir := range dirs {
		dirPaths = append(dirPaths, dir)
	}
	sort.Strings(dirPaths)
	for _, dir := range dirPaths {
		entries = append(entries, Entry{Name: pathBase(dir), Path: dir, Dir: true})
	}
	entries = append(entries, files...)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].Dir && !entries[j].Dir
		}
		return entries[i].Path < entries[j].Path
	})
	return entries
}

func directChildren(entries []Entry, folder string) []Entry {
	var out []Entry
	for _, entry := range entries {
		if pathParent(entry.Path) == folder {
			out = append(out, entry)
		}
	}
	return out
}

func childDirs(order []string, byPath func(id string) string, folder string) []string {
	seen := map[string]bool{}
	var out []string
	for _, id := range order {
		path := byPath(id)
		if path == "" {
			path = id
		}
		rest := path
		if folder != "" {
			if !strings.HasPrefix(path, folder+"/") {
				continue
			}
			rest = strings.TrimPrefix(path, folder+"/")
		}
		if !strings.Contains(rest, "/") {
			continue
		}
		seg, _, _ := strings.Cut(rest, "/")
		if seg == "" || seen[seg] {
			continue
		}
		seen[seg] = true
		out = append(out, seg)
	}
	return out
}
