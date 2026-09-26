package report

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Format is one diagnostic rendering. The zero value is unset.
// It is a cmd.Enum, so cmd.EnumArg[Format] parses a flag when a caller opts in.
type Format int

const (
	FormatText Format = iota + 1
	FormatTable
	FormatSARIF
	FormatRustc
)

// WriteFinding writes one diagnostic line:
// file:line:col: level: ruleId: message
func WriteFinding(w io.Writer, f Finding) error {
	fixNote := ""
	if f.Fixable && f.FixSkipped {
		fixNote = " [fix skipped: overlap]"
	} else if f.Fixable {
		fixNote = " [fixable]"
	}
	_, err := fmt.Fprintf(w, "%s:%d:%d: %s: %s: %s%s\n",
		f.File, f.Line, f.Column, f.Level, f.RuleID, f.Message, fixNote)
	return err
}

// WriteText writes findings as one diagnostic line each.
func WriteText(w io.Writer, findings []Finding) error {
	for _, f := range findings {
		if err := WriteFinding(w, f); err != nil {
			return err
		}
	}
	return nil
}

// WriteTable writes findings as an aligned table.
// Columns are LOCATION, LEVEL, RULE, MESSAGE, and FIX.
// LOCATION is file:line:col (1-based).
func WriteTable(w io.Writer, findings []Finding) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "LOCATION\tLEVEL\tRULE\tMESSAGE\tFIX"); err != nil {
		return err
	}
	for _, f := range findings {
		fix := ""
		if f.Fixable && f.FixSkipped {
			fix = "skipped"
		} else if f.Fixable {
			fix = "yes"
		}
		loc := fmt.Sprintf("%s:%d:%d", f.File, f.Line, f.Column)
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			loc, f.Level, f.RuleID, f.Message, fix); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// String is the token for cmd.EnumArg.
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatTable:
		return "table"
	case FormatSARIF:
		return "sarif"
	case FormatRustc:
		return "rustc"
	default:
		return ""
	}
}

// Values lists formats for cmd.EnumArg. The zero value is absent.
func (Format) Values() []Format {
	return []Format{FormatText, FormatTable, FormatSARIF, FormatRustc}
}

// ParseFormat accepts text, table, sarif, or rustc, in any case. An empty string is an error.
func ParseFormat(format string) (Format, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return FormatText, nil
	case "table":
		return FormatTable, nil
	case "sarif":
		return FormatSARIF, nil
	case "rustc":
		return FormatRustc, nil
	default:
		return 0, fmt.Errorf("%w %s (want text, table, sarif, or rustc)", ErrFormat, strconv.Quote(format))
	}
}

// Render writes findings in this format.
// root resolves relative paths for SARIF fixes and rustc file links.
// A zero Tool uses the lewkit driver name in SARIF.
// The zero Format returns ErrFormat and writes nothing.
func (f Format) Render(w io.Writer, root string, tool Tool, findings []Finding, rules []Rule) error {
	switch f {
	case FormatText:
		return WriteText(w, findings)
	case FormatTable:
		return WriteTable(w, findings)
	case FormatSARIF:
		return WriteSARIF(w, root, tool, findings, rules)
	case FormatRustc:
		return WriteRustc(w, root, findings)
	default:
		return fmt.Errorf("%w %s (want text, table, sarif, or rustc)", ErrFormat, strconv.Quote(f.String()))
	}
}
