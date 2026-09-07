package text

import "sort"

// LineIndex maps source byte offsets to 1-based line numbers and 0-based
// columns. Build once per source buffer; lookups are O(log lines).
//
// Offsets are UTF-8 byte offsets, matching tree-sitter StartByte/EndByte.
// Columns are also byte offsets from the start of the line (tree-sitter
// column semantics), not rune or display columns.
type LineIndex struct {
	// starts[i] is the byte offset where 1-based line i+1 begins.
	starts []int
}

// NewLineIndex builds a line index for text.
func NewLineIndex(text string) *LineIndex {
	// ~1 entry per 40 bytes (typical source line); grows if denser.
	starts := make([]int, 1, len(text)/40+2)
	starts[0] = 0
	for i := 0; i < len(text); i++ {
		if text[i] == '\n' && i+1 < len(text) {
			starts = append(starts, i+1)
		}
	}
	return &LineIndex{starts: starts}
}

// NewLineIndexBytes builds a line index for a byte buffer (UTF-8 source).
func NewLineIndexBytes(src []byte) *LineIndex {
	starts := make([]int, 1, len(src)/40+2)
	starts[0] = 0
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' && i+1 < len(src) {
			starts = append(starts, i+1)
		}
	}
	return &LineIndex{starts: starts}
}

// LineAt returns the 1-based line containing byte offset off.
// Offsets past the end map to the last line; empty or nil indexes return 1.
// Negative offsets map to line 1.
func (li *LineIndex) LineAt(off int) int {
	line, _ := li.LineColumnAt(off)
	return line
}

// LineAtU32 is LineAt for tree-sitter byte offsets.
func (li *LineIndex) LineAtU32(off uint32) int {
	return li.LineAt(int(off))
}

// LineColumnAt returns the 1-based line and 0-based byte column for off.
// Column is off minus the byte offset where that line starts.
// Offsets past the end map to the last line (column may exceed line length).
// Empty or nil indexes return (1, 0). Negative offsets return (1, 0).
func (li *LineIndex) LineColumnAt(off int) (line, column int) {
	if li == nil || len(li.starts) == 0 {
		return 1, 0
	}
	if off < 0 {
		return 1, 0
	}
	// largest i with starts[i] <= off
	i := sort.Search(len(li.starts), func(i int) bool {
		return li.starts[i] > off
	}) - 1
	if i < 0 {
		return 1, 0
	}
	return i + 1, off - li.starts[i]
}

// LineColumnAtU32 is LineColumnAt for tree-sitter byte offsets.
func (li *LineIndex) LineColumnAtU32(off uint32) (line, column int) {
	return li.LineColumnAt(int(off))
}
