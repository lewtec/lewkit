package cmd

import "github.com/lewtec/lewkit/x/table"

// FormatArg is the --format flag. Value is the format the process prints with.
type FormatArg struct {
	EnumArg[table.Format]
}

var (
	_ Parser            = (*FormatArg)(nil)
	_ Arg[table.Format] = (*FormatArg)(nil)
	_ ArgChooser        = FormatArg{}
)
