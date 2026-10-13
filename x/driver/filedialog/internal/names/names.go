// Package names keeps one display name per file-dialog entry.
package names

import (
	"path"
	"strconv"
	"strings"
)

// Take returns a display name that is not already in seen.
// A slash becomes an underscore. An empty name, ".", and ".." become "_".
// A collision is "stem (n)" plus the extension, starting at 2.
func Take(name string, seen map[string]struct{}) string {
	name = Clean(name)
	if _, ok := seen[name]; !ok {
		seen[name] = struct{}{}
		return name
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		next := stem + " (" + strconv.Itoa(n) + ")" + ext
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}
		return next
	}
}

// Clean maps a file-dialog name to a single path segment.
func Clean(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." || name == ".." {
		return "_"
	}
	return name
}
