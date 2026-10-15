package sql

import "io"

// File is one SQL statement log.
type File struct{}

// NewFile returns an empty statement log.
func NewFile() *File {
	return &File{}
}

// Take reads the artifact into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, errScaffold
	}
	return nil, errScaffold
}

// Make writes the file.
func (f *File) Make(w io.Writer) error {
	if f == nil || w == nil {
		return errScaffold
	}
	return errScaffold
}
