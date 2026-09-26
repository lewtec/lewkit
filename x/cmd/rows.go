package cmd

import (
	"context"
	"io"
	"iter"

	"github.com/lewtec/lewkit/x/table"
)

// Rows writes seq using the process --format value. A missing format is table.
// Every command that prints records calls Rows, so one flag selects the format.
func Rows[T any](ctx context.Context, w io.Writer, seq iter.Seq[T]) error {
	format, ok := Lookup[table.Format](ctx, "format")
	if !ok || format == 0 {
		format = table.Table
	}
	return table.Write(w, format, seq)
}
