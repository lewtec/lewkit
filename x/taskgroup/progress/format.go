package progress

import (
	"bytes"
	"os"

	"github.com/charmbracelet/x/term"
	"github.com/lewtec/lewkit/x/taskgroup"
)

// Format renders nodes as the progress tree (├ └ │ and status glyphs).
// width 0 uses the stdout terminal width, or 80 when stdout is not a terminal.
func Format(nodes []taskgroup.Node, width int) string {
	if width <= 0 {
		width = stdoutWidth()
	}
	var buf bytes.Buffer
	writeRows(&buf, nodes, width)
	return buf.String()
}

func stdoutWidth() int {
	if os.Stdout == nil {
		return defaultTermWidth
	}
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil || w <= 0 {
		return defaultTermWidth
	}
	return w
}

func writeRows(buf *bytes.Buffer, nodes []taskgroup.Node, width int) {
	for _, r := range layout(nodes) {
		buf.WriteString(formatRow(r, width))
		buf.WriteByte('\n')
	}
}
