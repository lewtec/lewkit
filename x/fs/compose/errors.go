package compose

import (
	"errors"
	iofs "io/fs"
)

var (
	// ErrType is an unknown type, or two types on one path.
	ErrType = errors.New("bad file type")
	// ErrSlot is a slot whose kind is unknown, or the same key with a different body.
	ErrSlot = errors.New("slot conflict")
	// ErrArity is a text or ref file that does not have exactly one slot.
	ErrArity = errors.New("file value count")
	// ErrData is a structured value that cannot be merged or encoded.
	ErrData = errors.New("structured value conflict")
	// ErrMode is two explicit modes that differ.
	ErrMode = errors.New("file mode conflict")
	// ErrPath is a destination name that cannot be stored.
	ErrPath = errors.New("invalid file path")
	// ErrRef is a ref slot that cannot be opened.
	ErrRef = errors.New("ref slot")
	// ErrMount is a CUE mount path that is not a dotted identifier chain.
	ErrMount = errors.New("invalid cue mount path")
	// ErrNilTree is Add or Merge on a nil tree.
	ErrNilTree = errors.New("nil tree")
	// ErrNilFilesystem is Squash on a nil filesystem.
	ErrNilFilesystem = errors.New("nil filesystem")
	// ErrNilCue is Constrain on a cue value with no context.
	ErrNilCue = errors.New("nil cue context")
)

func pathError(op, name string, err error) error {
	return &iofs.PathError{Op: op, Path: name, Err: err}
}
