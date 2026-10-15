package sql

import (
	"fmt"
	"io"
)

// note is the leading comment on a statement, without the "-- " marks.
type note struct {
	Comment string
}

func (n *note) setComment(text string) { n.Comment = text }

// Statement is one entry in a SQL log.
// The concrete types are CreateTable, AlterTable, DropTable, CreateIndex,
// Select, Insert, Update, and Delete.
type Statement interface {
	Kind() Kind
	statement()
}

// File is one SQL statement log.
type File struct {
	// Dialect is sqlite unless Postgres was selected or Take read another.
	Dialect Dialect
	// Statements is the log in order.
	Statements []Statement
}

// NewFile returns an empty sqlite log.
func NewFile() *File {
	return &File{Dialect: SQLite}
}

// SQLite sets the file dialect.
func (f *File) SQLite() *File {
	if f != nil {
		f.Dialect = SQLite
	}
	return f
}

// Postgres sets the file dialect.
func (f *File) Postgres() *File {
	if f != nil {
		f.Dialect = Postgres
	}
	return f
}

// Take reads a statement log.
// A missing dialect comment means sqlite.
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader", ErrInvalid)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return parse(string(body))
}

// Make writes the log.
// The first line is a dialect comment so a later Take restores Dialect.
func (f *File) Make(w io.Writer) error {
	if f == nil || w == nil {
		return fmt.Errorf("%w: file", ErrInvalid)
	}
	var b []byte
	text, err := formatFile(f)
	if err != nil {
		return err
	}
	b = []byte(text)
	_, err = w.Write(b)
	return err
}

func setComment(stmt Statement, text string) {
	if text == "" {
		return
	}
	type noted interface{ setComment(string) }
	if n, ok := stmt.(noted); ok {
		n.setComment(text)
	}
}
