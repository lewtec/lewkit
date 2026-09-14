// Package fs is shared errors and helpers for x/fs/* adapters.
//
// Adapters take an [io.Reader] and call [ReaderAt]. A missing
// [io.ReaderAt] is [ErrNeedReadAt]. They do not spool.
package fs

import (
	"errors"
	"io"
	iofs "io/fs"
)

// ErrNeedReadAt is [ReaderAt] on a reader that is not an [io.ReaderAt].
var ErrNeedReadAt = errors.New("need io.ReaderAt")

// ReaderAt returns r as an [io.ReaderAt], or [ErrNeedReadAt].
func ReaderAt(op string, r io.Reader) (io.ReaderAt, error) {
	ra, ok := r.(io.ReaderAt)
	if !ok {
		return nil, &iofs.PathError{Op: op, Path: "", Err: ErrNeedReadAt}
	}
	return ra, nil
}
