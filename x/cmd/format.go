package cmd

import "github.com/lewtec/lewkit/x/text/table"

// Output is the record printer. Flatten-embed it on a command that prints
// rows so --format and --columns land on that command.
type Output struct {
	Format  FormatArg `long:"format" default:"table" help:"output format" ctx:"format"`
	Columns StringArg `long:"columns" default:"" help:"columns as name or name=format" ctx:"columns"`
}

// FormatArg is the --format flag. Value is the format that command prints with.
type FormatArg struct {
	EnumArg[table.Format]
}

var (
	_ Parser            = (*FormatArg)(nil)
	_ Arg[table.Format] = (*FormatArg)(nil)
	_ ArgChooser        = FormatArg{}
)
