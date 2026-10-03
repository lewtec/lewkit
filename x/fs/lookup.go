package fs

import (
	iofs "io/fs"
	"strings"
)

// Entry is one record a filesystem already stores.
// [Lookup] reads children as it walks and does not keep them.
type Entry interface {
	// Name is the base name.
	Name() string
	// IsDir reports a directory.
	IsDir() bool
}

// Lookup walks name from root. read lists one directory and its result is not stored.
// "." and "" are root. The first child with a matching name wins.
// A missing child is [io/fs.ErrNotExist].
// A non-directory before the leaf is [io/fs.ErrInvalid].
func Lookup[E Entry](op, name string, root E, read func(E) ([]E, error)) (E, error) {
	var zero E
	if name == "." || name == "" {
		return root, nil
	}
	if err := CheckName(op, name); err != nil {
		return zero, err
	}
	cur := root
	parts := strings.Split(name, "/")
	for i, p := range parts {
		if !cur.IsDir() {
			prefix := strings.Join(parts[:i], "/")
			return zero, &iofs.PathError{Op: op, Path: prefix, Err: iofs.ErrInvalid}
		}
		kids, err := read(cur)
		if err != nil {
			return zero, err
		}
		var next E
		found := false
		for _, k := range kids {
			if k.Name() == p {
				next = k
				found = true
				break
			}
		}
		if !found {
			return zero, &iofs.PathError{Op: op, Path: name, Err: iofs.ErrNotExist}
		}
		cur = next
	}
	return cur, nil
}
