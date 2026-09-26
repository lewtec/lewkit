package report

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()
	got, err := ParseLevel(" ERROR ")
	require.NoError(t, err)
	require.Equal(t, LevelError, got)
	got, err = ParseLevel("")
	require.NoError(t, err)
	require.Equal(t, LevelWarning, got)
	_, err = ParseLevel("fatal")
	require.Error(t, err)
}

func TestNormalizeFormat(t *testing.T) {
	t.Parallel()
	got, err := NormalizeFormat("RUSTc")
	require.NoError(t, err)
	require.Equal(t, "rustc", got)
	_, err = NormalizeFormat("")
	require.ErrorIs(t, err, ErrFormat)
	_, err = NormalizeFormat("html")
	require.Error(t, err)
}

func TestWriteTextAndTable(t *testing.T) {
	t.Parallel()
	f := Finding{
		RuleID:  "demo/hello",
		Level:   LevelWarning,
		Message: "don't greet",
		File:    "main.go",
		Line:    3,
		Column:  6,
		Fixable: true,
	}
	var text bytes.Buffer
	require.NoError(t, WriteFormat(&text, "text", "", Tool{}, []Finding{f}, nil))
	require.Equal(t, "main.go:3:6: warning: demo/hello: don't greet [fixable]\n", text.String())

	var table bytes.Buffer
	require.NoError(t, WriteTable(&table, []Finding{f}))
	got := table.String()
	for _, part := range []string{"LOCATION", "main.go:3:6", "warning", "demo/hello", "yes"} {
		require.Contains(t, got, part)
	}
}

func TestSpanLocAndOverlap(t *testing.T) {
	t.Parallel()
	src := []byte("func Hello() {}\n")
	line, col, endLine, endCol, snip, err := SpanLoc(src, Span{StartByte: 5, EndByte: 10})
	require.NoError(t, err)
	require.Equal(t, 1, line)
	require.Equal(t, 6, col)
	require.Equal(t, 1, endLine)
	require.Equal(t, 11, endCol)
	require.Equal(t, "Hello", snip)
	_, _, _, _, _, err = SpanLoc(src, Span{StartByte: 0, EndByte: 99})
	require.Error(t, err)
	edits := []Edit{{StartByte: 5, EndByte: 10}}
	require.True(t, EditsOverlap(edits, []Span{{StartByte: 8, EndByte: 12}}))
	require.False(t, EditsOverlap(edits, []Span{{StartByte: 10, EndByte: 12}}))
	require.Equal(t, "a…", OneLine(" a\nb "))
}
