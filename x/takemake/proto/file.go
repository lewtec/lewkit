package proto

import (
	"io"

	"github.com/lewtec/lewkit/x/errs"
)

// File is one proto file.
type File struct{}

// NewFile returns an empty proto file.
func NewFile() *File {
	return &File{}
}

// Take reads the artifact into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, errs.ErrScaffold
	}
	return nil, errs.ErrScaffold
}

// Make writes the file.
func (f *File) Make(w io.Writer) error {
	if f == nil || w == nil {
		return errs.ErrScaffold
	}
	return errs.ErrScaffold
}
