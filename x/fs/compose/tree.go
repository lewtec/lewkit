package compose

import (
	"fmt"
	iofs "io/fs"
	"iter"
	"maps"
	"slices"

	"github.com/lewtec/lewkit/x/path"
)

// Tree is a destination. Add and Merge unify files into it.
type Tree struct {
	files map[string]File
	// descendants maps a strict ancestor to one stored path under it.
	descendants map[string]string
}

// New returns an empty tree.
func New() *Tree {
	return &Tree{files: map[string]File{}}
}

// All yields each declaration in path order.
// The yielded [File] is a copy.
func (tree *Tree) All() iter.Seq2[path.Path, File] {
	return func(yield func(path.Path, File) bool) {
		if tree == nil {
			return
		}
		for _, name := range slices.Sorted(maps.Keys(tree.files)) {
			if !yield(path.New(name), snapshot(tree.files[name])) {
				return
			}
		}
	}
}

func snapshot(file File) File {
	file.Values = maps.Clone(file.Values)
	file.Data = cloneMap(file.Data)
	return file
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
	if other, conflicts := tree.conflictingPath(name); conflicts {
		return pathError("merge", key, fmt.Errorf("%w: %s", ErrPath, other))
	}
	current, exists := tree.files[key]
	if !exists {
		if tree.files == nil {
			tree.files = map[string]File{}
		}
		tree.files[key] = normalized
		tree.recordAncestors(name)
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

func (tree *Tree) conflictingPath(name path.Path) (string, bool) {
	for parent := name.Parent(); parent.String() != "."; parent = parent.Parent() {
		key := parent.String()
		if _, exists := tree.files[key]; exists {
			return key, true
		}
	}
	if other, exists := tree.descendants[name.String()]; exists {
		return other, true
	}
	return "", false
}

func (tree *Tree) recordAncestors(name path.Path) {
	stored := name.String()
	if tree.descendants == nil {
		tree.descendants = map[string]string{}
	}
	for parent := name.Parent(); parent.String() != "."; parent = parent.Parent() {
		key := parent.String()
		if _, exists := tree.descendants[key]; exists {
			return
		}
		tree.descendants[key] = stored
	}
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
