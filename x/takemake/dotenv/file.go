package dotenv

import (
	"io"

	"github.com/lewtec/lewkit/x/errs"
	syntax "github.com/lewtec/lewkit/x/parse/dotenv"
)

var (
	ErrNilReader     = errs.Wrap(errs.ErrInvalid, "dotenv: nil reader")
	ErrNilWriter     = errs.Wrap(errs.ErrInvalid, "dotenv: nil writer")
	ErrNilFile       = errs.Wrap(errs.ErrInvalid, "dotenv: nil file")
	ErrNilAssignment = errs.Wrap(errs.ErrInvalid, "dotenv: nil assignment")
	ErrLine          = syntax.ErrLine
	ErrKey           = errs.Wrap(errs.ErrEmpty, "dotenv: empty key")
	ErrEncode        = errs.Wrap(errs.ErrInvalid, "dotenv: quote cannot represent the value")
)

// Assignment is one entry in file order.
// Comment is the note without hash marks. Newlines in Comment are
// separate comment lines. An empty Key is a note with no name.
type Assignment struct {
	Key     string
	Value   string
	Quote   Quote
	Comment string
	Export  bool
}

// File is assignments in order.
type File struct {
	Assignments []*Assignment
}

// New returns an empty file.
func New() *File {
	return &File{}
}

// Assign appends a bare assignment and returns the file.
func (f *File) Assign(key, value string) *File {
	if f == nil {
		return nil
	}
	f.Assignments = append(f.Assignments, &Assignment{
		Key:   key,
		Value: value,
		Quote: QuoteBare,
	})
	return f
}

// Comment sets the note on the latest assignment.
// On an empty file it appends a nameless note.
func (f *File) Comment(text string) *File {
	if f == nil {
		return nil
	}
	if len(f.Assignments) == 0 {
		f.Assignments = append(f.Assignments, &Assignment{Comment: text})
		return f
	}
	f.Assignments[len(f.Assignments)-1].Comment = text
	return f
}

// Bare writes the latest value without quotes.
func (f *File) Bare() *File {
	return f.setQuote(QuoteBare)
}

// Single writes the latest value in single quotes.
func (f *File) Single() *File {
	return f.setQuote(QuoteSingle)
}

// Double writes the latest value in double quotes.
func (f *File) Double() *File {
	return f.setQuote(QuoteDouble)
}

// Export marks the latest assignment as an export.
func (f *File) Export() *File {
	if last := f.last(); last != nil {
		last.Export = true
	}
	return f
}

func (f *File) setQuote(quote Quote) *File {
	if last := f.last(); last != nil {
		last.Quote = quote
	}
	return f
}

func (f *File) last() *Assignment {
	if f == nil || len(f.Assignments) == 0 {
		return nil
	}
	return f.Assignments[len(f.Assignments)-1]
}

// Take reads dotenv text into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(string(raw))
}

// Make writes the file.
func (f *File) Make(w io.Writer) error {
	if f == nil {
		return ErrNilFile
	}
	if w == nil {
		return ErrNilWriter
	}
	for _, assignment := range f.Assignments {
		if err := writeAssignment(w, assignment); err != nil {
			return err
		}
	}
	return nil
}
