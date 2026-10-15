package zip

import "io"

// File is the document this package round-trips.
type File struct{}

// New returns an empty file.
func New() *File {
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
