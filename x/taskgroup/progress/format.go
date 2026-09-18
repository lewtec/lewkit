package progress

import (
	"bytes"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// Format renders nodes as the progress tree (├ └ │ and status glyphs).
// width 0 uses the default terminal width.
func Format(nodes []taskgroup.Node, width int) string {
	if width <= 0 {
		width = defaultTermWidth
	}
	var buf bytes.Buffer
	for _, r := range layout(nodes) {
		buf.WriteString(formatRow(r, width))
		buf.WriteByte('\n')
	}
	return buf.String()
}
