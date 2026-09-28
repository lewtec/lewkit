// Package common holds shared helpers for ephemeral Android, iOS, and macOS host trees.
package common

import "errors"

// Config / path sentinels used by ios and mac generators.
var (
	ErrConfigPathEmpty = errors.New("config path is empty")
	ErrGoMainNotDir    = errors.New("go_main must be a directory (main package)")
	ErrOutDirRequired  = errors.New("out dir is required")
	ErrOutPathNotDir   = errors.New("out path exists and is not a directory")
	ErrOutDirNotEmpty  = errors.New("out dir is not empty (use --force)")
)
