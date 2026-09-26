// Package report writes diagnostics as a text line, a table, a rustc-style
// snippet, or a SARIF 2.1.0 log.
package report

// Finding is one diagnostic row for every output format.
type Finding struct {
	RuleID  string
	Level   Level
	Message string
	File    string
	Line    int // 1-based
	Column  int // 1-based
	EndLine int
	EndCol  int
	Snippet string
	// Edits are replacement edits for this match (SARIF result.fixes).
	Edits []Edit
	// Fixable is true when edits were produced.
	Fixable bool
	// FixSkipped is true when Fixable but edits were dropped because of a conflict.
	FixSkipped bool
	// Source is optional file bytes at match time. SARIF and rustc use it for regions.
	Source []byte
}

// Rule is a reporting descriptor for SARIF tool.driver.rules.
type Rule struct {
	ID      string
	Message string
	Level   Level
}

// Edit replaces the half-open byte span [StartByte, EndByte) with NewText.
type Edit struct {
	File      string
	StartByte uint32
	EndByte   uint32
	NewText   string
}

// Span returns the half-open byte range of e.
func (e Edit) Span() Span {
	return Span{StartByte: e.StartByte, EndByte: e.EndByte}
}

// Tool names the SARIF driver. Zero fields use lewkit defaults.
type Tool struct {
	Name           string
	Version        string
	InformationURI string
}
