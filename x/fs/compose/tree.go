package compose

import (
	"fmt"
	iofs "io/fs"
	"maps"

	"github.com/lewtec/lewkit/x/path"
)

// Tree is a destination. Add and Merge unify files into it.
type Tree struct {
	files map[string]File
}

// New returns an empty tree.
func New() *Tree {
	return &Tree{files: map[string]File{}}
}

// Add merges one file into the tree.
func (tree *Tree) Add(name path.Path, file File) error {
	if tree == nil {
		return errNilTree
	}
	if err := checkName(name); err != nil {
		return pathError("merge", name.String(), err)
	}
	normalized, err := file.normalized()
	if err != nil {
		return pathError("merge", name.String(), err)
	}
	if err := normalized.check(); err != nil {
		return pathError("merge", name.String(), err)
	}
	key := name.String()
	for existing := range tree.files {
		if pathClash(path.New(existing), name) {
			return pathError("merge", key, fmt.Errorf("%w: %s", ErrPath, existing))
		}
	}
	current, exists := tree.files[key]
	if !exists {
		if tree.files == nil {
			tree.files = map[string]File{}
		}
		tree.files[key] = normalized
		return nil
	}
	merged, err := mergeFile(current, normalized)
	if err != nil {
		return pathError("merge", key, err)
	}
	tree.files[key] = merged
	return nil
}

// Merge unifies other into tree. A nil other is a no-op.
// Merge is commutative when both sides agree.
func (tree *Tree) Merge(other *Tree) error {
	if other == nil || len(other.files) == 0 {
		return nil
	}
	if tree == nil {
		return errNilTree
	}
	for name, file := range maps.Clone(other.files) {
		if err := tree.Add(path.New(name), file); err != nil {
			return err
		}
	}
	return nil
}

func mergeFile(current, extra File) (File, error) {
	if current.Type != extra.Type {
		return File{}, fmt.Errorf("%w: %s vs %s", ErrType, current.Type, extra.Type)
	}
	mode, err := mergeMode(current.Mode, extra.Mode)
	if err != nil {
		return File{}, err
	}
	merged := File{Type: current.Type, Mode: mode}
	if current.Type.structured() {
		data, err := mergeData(current.Data, extra.Data)
		if err != nil {
			return File{}, err
		}
		merged.Data = data
		return merged, nil
	}
	values, err := mergeSlots(current.Values, extra.Values)
	if err != nil {
		return File{}, err
	}
	merged.Values = values
	if err := merged.check(); err != nil {
		return File{}, err
	}
	return merged, nil
}

func mergeMode(current, extra iofs.FileMode) (iofs.FileMode, error) {
	if current != 0 && extra != 0 && current != extra {
		return 0, ErrMode
	}
	if current == 0 {
		return extra, nil
	}
	return current, nil
}

func mergeSlots(current, extra map[string]Slot) (map[string]Slot, error) {
	out := maps.Clone(current)
	if out == nil {
		out = make(map[string]Slot, len(extra))
	}
	for key, slot := range extra {
		existing, exists := out[key]
		if exists && existing != slot {
			return nil, fmt.Errorf("%w: %s", ErrSlot, key)
		}
		out[key] = slot
	}
	return out, nil
}
