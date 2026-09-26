package report

import (
	"strings"

	"github.com/lewtec/lewkit/x/text"
)

// Span is a half-open byte range in one source buffer.
type Span struct {
	StartByte uint32
	EndByte   uint32
}

// Empty reports a zero-width span.
func (s Span) Empty() bool { return s.StartByte == s.EndByte }

// Overlaps reports whether the half-open ranges share a byte.
func Overlaps(a, b Span) bool {
	return a.StartByte < b.EndByte && b.StartByte < a.EndByte
}

// SpanLoc maps a half-open byte span to 1-based line and column and a one-line snippet.
func SpanLoc(src []byte, sp Span) (line, col, endLine, endCol int, snippet string, err error) {
	if int(sp.EndByte) > len(src) || sp.StartByte > sp.EndByte {
		return 0, 0, 0, 0, "", ErrSpan
	}
	li := text.NewLineIndexBytes(src)
	l, c0 := li.LineColumnAtU32(sp.StartByte)
	el, ec0 := li.LineColumnAtU32(sp.EndByte)
	snip := string(src[sp.StartByte:sp.EndByte])
	if i := strings.IndexByte(snip, '\n'); i >= 0 {
		snip = snip[:i] + "…"
	}
	return l, c0 + 1, el, ec0 + 1, snip, nil
}

// OneLine collapses s to a single display line. A multi-line value keeps the first line and an ellipsis.
func OneLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i]) + "…"
	}
	return s
}

// EditsOverlap reports whether any edit span overlaps claimed.
func EditsOverlap(edits []Edit, claimed []Span) bool {
	for _, e := range edits {
		sp := e.Span()
		for _, c := range claimed {
			if Overlaps(sp, c) {
				return true
			}
		}
	}
	return false
}
