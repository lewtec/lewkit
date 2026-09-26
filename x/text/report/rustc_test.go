package report

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriteRustcSnippetUnderline(t *testing.T) {
	t.Parallel()
	src := []byte("package main\n\nfunc Hello() {}\n")
	f := Finding{
		RuleID:  "demo/hello",
		Level:   LevelError,
		Message: "don't greet",
		File:    "main.go",
		Line:    3,
		Column:  6,
		EndLine: 3,
		EndCol:  11,
		Snippet: "Hello",
		Source:  src,
	}
	var buf bytes.Buffer
	require.NoError(t, newRustcWriter(&buf, "", false).write([]Finding{f}))
	want := "" +
		"error[demo/hello]: don't greet\n" +
		" --> main.go:3:6\n" +
		"2 | \n" +
		"3 | func \x1b[4mHello\x1b[24m() {}\n"
	require.Equal(t, want, buf.String())
}

func TestWriteRustcExclusiveNewlineEnd(t *testing.T) {
	t.Parallel()
	src := []byte("import unused \"fmt\"\n\nfunc main() {}\n")
	f := Finding{
		RuleID:  "imports/unused-named",
		Level:   LevelWarning,
		Message: "unused named import",
		File:    "main.go",
		Line:    1,
		Column:  1,
		EndLine: 2,
		EndCol:  1,
		Source:  src,
	}
	var buf bytes.Buffer
	require.NoError(t, WriteRustc(&buf, "", []Finding{f}))
	got := buf.String()
	require.Contains(t, got, "1 | \x1b[4mimport unused \"fmt\"\x1b[24m")
	require.NotContains(t, got, "2 | \x1b[4m")
	require.NotContains(t, got, "3 |")
}

func TestWriteRustcSnippetFallback(t *testing.T) {
	t.Parallel()
	f := Finding{
		RuleID:  "x",
		Level:   LevelWarning,
		Message: "here",
		File:    "a.go",
		Line:    12,
		Column:  1,
		EndLine: 12,
		EndCol:  4,
		Snippet: "foo",
	}
	var buf bytes.Buffer
	require.NoError(t, WriteRustc(&buf, "", []Finding{f}))
	got := buf.String()
	require.Contains(t, got, "warning[x]: here")
	require.Contains(t, got, "12 | \x1b[4mfoo\x1b[24m")
}

func TestWriteRustcFixDiff(t *testing.T) {
	t.Parallel()
	src := []byte("func Hello() {}\n")
	f := Finding{
		Level:   LevelNote,
		Message: "rename",
		File:    "a.go",
		Line:    1,
		Column:  6,
		EndLine: 1,
		EndCol:  11,
		Fixable: true,
		Source:  src,
		Edits: []Edit{{
			File:      "a.go",
			StartByte: 5,
			EndByte:   10,
			NewText:   "Bye",
		}},
	}
	var buf bytes.Buffer
	require.NoError(t, newRustcWriter(&buf, "", false).write([]Finding{f}))
	got := buf.String()
	require.Contains(t, got, "help: fix available")
	require.Contains(t, got, "- func Hello() {}")
	require.Contains(t, got, "+ func Bye() {}")
}
