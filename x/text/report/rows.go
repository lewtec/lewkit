package report

import (
	"fmt"
	"io"
	"iter"

	"github.com/lewtec/lewkit/x/text/table"
)

// findingRow is one finding as record columns. The same view feeds the
// aligned table, JSONL, and CSV.
type findingRow struct {
	Location string `json:"LOCATION"`
	Level    string `json:"LEVEL"`
	Rule     string `json:"RULE"`
	Message  string `json:"MESSAGE"`
	Fix      string `json:"FIX"`
}

type findingSpec struct {
	Location table.Field[string]
	Level    table.Field[string]
	Rule     table.Field[string]
	Message  table.Field[string]
	Fix      table.Field[string]
}

var findingLayout = table.Must(table.Make[findingRow](findingSpec{}))

// WriteTable writes findings as an aligned table.
// Columns are LOCATION, LEVEL, RULE, MESSAGE, and FIX.
// LOCATION is file:line:col (1-based).
func WriteTable(writer io.Writer, findings []Finding) error {
	return WriteRecords(writer, table.Table, findings)
}

// WriteRecords writes findings with the shared record writer.
// format is table, jsonl, or csv. The columns match WriteTable.
// A command that embeds cmd.Output passes that format here.
func WriteRecords(writer io.Writer, format table.Format, findings []Finding) error {
	return table.Write(writer, format, findingRows(findings), findingLayout)
}

func findingRows(findings []Finding) iter.Seq[findingRow] {
	return func(yield func(findingRow) bool) {
		for _, finding := range findings {
			if !yield(findingRecord(finding)) {
				return
			}
		}
	}
}

func findingRecord(finding Finding) findingRow {
	fix := ""
	if finding.Fixable && finding.FixSkipped {
		fix = "skipped"
	} else if finding.Fixable {
		fix = "yes"
	}
	return findingRow{
		Location: fmt.Sprintf("%s:%d:%d", finding.File, finding.Line, finding.Column),
		Level:    finding.Level.String(),
		Rule:     finding.RuleID,
		Message:  finding.Message,
		Fix:      fix,
	}
}
