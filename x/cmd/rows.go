package cmd

import (
	"context"
	"io"
	"iter"

	"github.com/lewtec/lewkit/x/text/table"
)

// Rows writes seq using the process --format value. A missing format is table.
// layout is the column list. Nil uses Formatter on T, then struct fields.
// --columns picks names and replaces formats: name or name=format.
// Every command that prints records calls Rows, so one flag selects the format.
func Rows[T any](ctx context.Context, w io.Writer, seq iter.Seq[T], layout table.Formatter[T]) error {
	format, ok := Lookup[table.Format](ctx, "format")
	if !ok || format == 0 {
		format = table.Table
	}
	cols, err := table.Resolve(layout)
	if err != nil {
		return err
	}
	spec, ok := Lookup[string](ctx, "columns")
	if !ok {
		spec = ""
	}
	cols, err = table.Select(cols, spec)
	if err == nil {
		err = table.Write(w, format, seq, table.Fields[T](cols))
	}
	return err
}
