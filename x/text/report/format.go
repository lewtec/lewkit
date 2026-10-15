package report

import (
	"fmt"
	"io"
)

// Format is one diagnostic rendering. The zero value is unset.
// It is a cmd.Enum, so cmd.EnumArg[Format] parses a flag when a caller opts in.
//
//lewkit:enum fold
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
		return fmt.Errorf("%w: %q", ErrFormat, f.String())
	}
}
