package golang

import "io"

// File is one Go source file.
type File struct {
	// Name is the package clause.
	Name string
}

// NewFile returns an empty file for package name.
func NewFile(name string) *File {
	return &File{Name: name}
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
