package cmd

import (
	"context"
	"io"
	"iter"

	"github.com/lewtec/lewkit/x/text/table"
)

// Rows writes sequence using the process --format value. A missing format is table.
// layout is the column list. Nil uses Formatter on Row, then struct fields.
// --columns picks names and replaces formats: name or name=format.
// Every command that prints records calls Rows, so one flag selects the format.
func Rows[Row any](ctx context.Context, writer io.Writer, sequence iter.Seq[Row], layout table.Formatter[Row]) error {
	format, ok := Lookup[table.Format](ctx, "format")
	if !ok || format == 0 {
		format = table.Table
	}
	columns, err := table.Resolve(layout)
	if err != nil {
		return err
	}
	spec, ok := Lookup[string](ctx, "columns")
	if !ok {
		spec = ""
	}
	columns, err = table.Select(columns, spec)
	if err == nil {
		err = table.Write(writer, format, sequence, table.Fields[Row](columns))
	}
	return err
}
