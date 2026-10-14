package compose

import (
	"errors"
	iofs "io/fs"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	// ErrType is an unknown type, or two types on one path.
	ErrType = errs.Wrap(errs.ErrInvalid, "bad file type")
	// ErrSlot is a slot whose kind is unknown, or the same key with a different body.
	ErrSlot = errors.New("slot conflict")
	// ErrArity is a text or ref file that does not have exactly one slot.
	ErrArity = errors.New("file value count")
	// ErrData is a structured value that cannot be merged or encoded.
	ErrData = errors.New("structured value conflict")
	// ErrMode is two explicit modes that differ.
	ErrMode = errors.New("file mode conflict")
	// ErrPath is a destination name that cannot be stored.
	ErrPath = errs.Wrap(errs.ErrInvalid, "invalid file path")
	// ErrRef is a ref slot that cannot be opened.
	ErrRef = errors.New("ref slot")
	// ErrMount is a CUE mount path that is not a dotted identifier chain.
	ErrMount = errs.Wrap(errs.ErrInvalid, "invalid cue mount path")
	// ErrRegistered is [Register] of a name that already has a format.
	ErrRegistered           = errs.Wrap(errs.ErrAlreadyExists, "format already registered")
	ErrCompileMount         = errors.New("compile mount")
	ErrConstrainUnifyMount  = errors.New("unify mount")
	ErrEncodeINI            = errors.New("")
	ErrEncodeSlotsValues    = errors.New("values")
	ErrFileCheckValues      = errors.New("values")
	ErrIniLine              = errors.New("")
	ErrMergeData            = errors.New("")
	ErrParseFileType        = errors.New("type")
	ErrParseFileValues      = errors.New("values")
	ErrParseFileValues2     = errors.New("values")
	ErrParseParse           = errors.New("parse")
	ErrParseSlotSlotKindRef = errors.New("slot kind ref")
	ErrSlotKindLink         = errors.New("slot kind link")
	ErrSlotKindText         = errors.New("slot kind text")
	ErrXmlWriterNode        = errors.New("")
)

var (
	errNilTree       = errors.New("nil tree")
	errNilFilesystem = errors.New("nil filesystem")
	errNilCue        = errors.New("nil cue context")
)

func pathError(op, name string, err error) error {
	return &iofs.PathError{Op: op, Path: name, Err: err}
}
