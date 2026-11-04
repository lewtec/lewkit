package text

import (
	"testing"
)

func TestLineAt(t *testing.T) {
	// "a\nbb\nccc\n" line starts: 0, 2, 5
	multi := NewLineIndex("a\nbb\nccc\n")
	// no trailing newline: "a\nb" line starts: 0, 2
	noNL := NewLineIndex("a\nb")
	single := NewLineIndex("hello")
	empty := NewLineIndex("")

	tests := []struct {
		name string
		li   *LineIndex
		off  int
		want int
	}{
		// multi-line: first / last byte per line, mid-line, newline byte, past EOF
		{name: "multi first byte", li: multi, off: 0, want: 1},
		{name: "multi newline of line1", li: multi, off: 1, want: 1},
		{name: "multi first of line2", li: multi, off: 2, want: 2},
		{name: "multi mid line2", li: multi, off: 3, want: 2},
		{name: "multi last of line2", li: multi, off: 4, want: 2},
		{name: "multi first of line3", li: multi, off: 5, want: 3},
		{name: "multi mid line3", li: multi, off: 6, want: 3},
		{name: "multi last content byte", li: multi, off: 7, want: 3},
		{name: "multi final newline", li: multi, off: 8, want: 3},
		{name: "multi past EOF", li: multi, off: 100, want: 3},
		{name: "multi one past end", li: multi, off: 9, want: 3},
		{name: "multi negative off", li: multi, off: -1, want: 1},

		// no trailing newline
		{name: "noNL first", li: noNL, off: 0, want: 1},
		{name: "noNL newline", li: noNL, off: 1, want: 1},
		{name: "noNL last byte", li: noNL, off: 2, want: 2},
		{name: "noNL past EOF", li: noNL, off: 3, want: 2},

		// single line
		{name: "single first", li: single, off: 0, want: 1},
		{name: "single mid", li: single, off: 2, want: 1},
		{name: "single last", li: single, off: 4, want: 1},
		{name: "single past EOF", li: single, off: 5, want: 1},

		// empty file: always line 1
		{name: "empty zero", li: empty, off: 0, want: 1},
		{name: "empty past EOF", li: empty, off: 1, want: 1},
		{name: "empty far past", li: empty, off: 100, want: 1},

		// nil receiver
		{name: "nil receiver", li: nil, off: 0, want: 1},
		{name: "nil past", li: nil, off: 99, want: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.li.LineAt(tt.off); got != tt.want {
				t.Errorf("LineAt(%d) = %d; want %d", tt.off, got, tt.want)
			}
		})
	}
}

func TestLineColumnAt(t *testing.T) {
	// "a\nbb\nccc" starts: 0, 2, 5
	li := NewLineIndex("a\nbb\nccc")

	tests := []struct {
		name       string
		off        int
		wantLine   int
		wantColumn int
	}{
		{name: "start", off: 0, wantLine: 1, wantColumn: 0},
		{name: "newline on line1", off: 1, wantLine: 1, wantColumn: 1},
		{name: "line2 start", off: 2, wantLine: 2, wantColumn: 0},
		{name: "line2 mid", off: 3, wantLine: 2, wantColumn: 1},
		{name: "line2 last", off: 4, wantLine: 2, wantColumn: 2},
		{name: "line3 start", off: 5, wantLine: 3, wantColumn: 0},
		{name: "line3 mid", off: 6, wantLine: 3, wantColumn: 1},
		{name: "line3 last", off: 7, wantLine: 3, wantColumn: 2},
		{name: "past EOF", off: 8, wantLine: 3, wantColumn: 3},
		{name: "negative", off: -1, wantLine: 1, wantColumn: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			line, col := li.LineColumnAt(tt.off)
			if line != tt.wantLine || col != tt.wantColumn {
				t.Errorf("LineColumnAt(%d) = (%d, %d); want (%d, %d)",
					tt.off, line, col, tt.wantLine, tt.wantColumn)
			}
		})
	}

	t.Run("nil", func(t *testing.T) {
		t.Parallel()
		var nilLI *LineIndex
		line, col := nilLI.LineColumnAt(5)
		if line != 1 || col != 0 {
			t.Errorf("nil LineColumnAt = (%d, %d); want (1, 0)", line, col)
		}
	})
}

func TestLineAtU32(t *testing.T) {
	li := NewLineIndex("a\nb\n")
	if got := li.LineAtU32(2); got != 2 {
		t.Errorf("LineAtU32(2) = %d; want 2", got)
	}
	if got := li.LineAtU32(100); got != 2 {
		t.Errorf("LineAtU32(100) = %d; want 2", got)
	}
	line, col := li.LineColumnAtU32(3)
	if line != 2 || col != 1 {
		t.Errorf("LineColumnAtU32(3) = (%d, %d); want (2, 1)", line, col)
	}
}

func TestNewLineIndexBytesMatchesString(t *testing.T) {
	src := []byte("x\ny\nz")
	fromBytes := NewLineIndexBytes(src)
	fromString := NewLineIndex(string(src))
	for off := 0; off <= len(src)+1; off++ {
		lb, cb := fromBytes.LineColumnAt(off)
		ls, cs := fromString.LineColumnAt(off)
		if lb != ls || cb != cs {
			t.Fatalf("off %d: bytes=(%d,%d) string=(%d,%d)", off, lb, cb, ls, cs)
		}
	}
}
