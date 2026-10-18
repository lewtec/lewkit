// Package common holds shared helpers for host build trees.
package common

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

// Config / path sentinels used by ios and mac generators.
var (
	ErrConfigPathEmpty    = errs.Wrap(errs.ErrEmpty, "config path is empty")
	ErrGoMainNotDir       = errors.New("go_main must be a directory (main package)")
	ErrOutDirRequired     = errs.Wrap(errs.ErrRequired, "out dir is required")
	ErrOutPathNotDir      = errs.Wrap(errs.ErrInvalid, "out path exists and is not a directory")
	ErrOutDirNotEmpty     = errs.Wrap(errs.ErrAlreadyExists, "out dir is not empty (use --force)")
	ErrWorkDirLeftAt      = errors.New("work dir left at")
	ErrXcodegenGenerate   = errors.New("xcodegen generate")
	ErrXcodebuild         = errors.New("xcodebuild")
	ErrOutAppRequired     = errs.Wrap(errs.ErrRequired, "out .app path is required (or use --go-only)")
	ErrXcodebuildNotFound = errs.Wrap(errs.ErrNotFound, "xcodebuild not found; install Xcode")
	ErrXcodeGenNotFound   = errs.Wrap(errs.ErrNotFound, "xcodegen not found; install xcodegen (mise/brew)")
	ErrIcons              = errors.New("icons")
)
