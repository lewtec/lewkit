package dotenv

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	syntax "github.com/lewtec/lewkit/x/parse/dotenv"
	"github.com/stretchr/testify/require"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	const src = "# database\r\n" +
		"export HOST = \"localhost\"\n" +
		"\n" +
		"PORT=5432 # primary\n" +
		"PASSWORD='hunter2'\n" +
		"LIT='a\\nb'\n" +
		"MSG=\"a\\nb\"\n" +
		"SAY=\"say \\\"hi\\\"\"\n" +
		"TOKEN=raw#hash\n" +
		"RAW=a\\nb\n" +
		"EMPTY=\n" +
		"BLANK=\"\"\n" +
		"Q=''\n" +
		"SP=\" a \"\n" +
		"COLOR=#fff\n" +
		"GAP= # note\n" +
		"DUP=1\n" +
		"DUP=2\n" +
		"# trailer\n"

	first, printed := takeMake(t, src)
	require.Equal(t, []view{
		{"HOST", "localhost", "database", syntax.QuoteDouble, true},
		{"PORT", "5432", "primary", syntax.QuoteBare, false},
		{"PASSWORD", "hunter2", "", syntax.QuoteSingle, false},
		{"LIT", `a\nb`, "", syntax.QuoteSingle, false},
		{"MSG", "a\nb", "", syntax.QuoteDouble, false},
		{"SAY", `say "hi"`, "", syntax.QuoteDouble, false},
		{"TOKEN", "raw#hash", "", syntax.QuoteBare, false},
		{"RAW", `a\nb`, "", syntax.QuoteBare, false},
		{"EMPTY", "", "", syntax.QuoteBare, false},
		{"BLANK", "", "", syntax.QuoteDouble, false},
		{"Q", "", "", syntax.QuoteSingle, false},
		{"SP", " a ", "", syntax.QuoteDouble, false},
		{"COLOR", "#fff", "", syntax.QuoteBare, false},
		{"GAP", "", "note", syntax.QuoteBare, false},
		{"DUP", "1", "", syntax.QuoteBare, false},
		{"DUP", "2", "trailer", syntax.QuoteBare, false},
	}, views(first))

	second, printedAgain := takeMake(t, printed)
	require.Equal(t, views(first), views(second))
	require.Equal(t, printed, printedAgain)
}

func TestCommentStaysOnAssignment(t *testing.T) {
	t.Parallel()
	first, _ := takeMake(t, "# above\nKEY=1 # inline\n")
	require.Equal(t, []view{
		{"KEY", "1", "above\ninline", syntax.QuoteBare, false},
	}, views(first))
}

func TestCommentOnly(t *testing.T) {
	t.Parallel()
	first, printed := takeMake(t, "# only\n# two\n")
	require.Equal(t, []view{
		{"", "", "only\ntwo", 0, false},
	}, views(first))
	require.Equal(t, "# only\n# two\n", printed)
}

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := New().
		Assign("HOST", "localhost").Double().Export().Comment("database").
		Assign("PORT", "5432").Comment("primary").
		Assign("MSG", "a\nb").Double().
		Assign("DUP", "1").
		Assign("DUP", "2").Comment("trailer")
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	printed := buf.String()
	taken, err := Take(strings.NewReader(printed))
	require.NoError(t, err)
	require.Equal(t, views(file), views(taken))
	var again bytes.Buffer
	require.NoError(t, taken.Make(&again))
	require.Equal(t, printed, again.String())
}

func TestAssignStoresTheChild(t *testing.T) {
	t.Parallel()
	file := New().Assign("A", "1").Comment("note").Single()
	require.Len(t, file.Assignments, 1)
	got := file.Assignments[0]
	require.Equal(t, "A", got.Key)
	require.Equal(t, "1", got.Value)
	require.Equal(t, "note", got.Comment)
	require.Equal(t, syntax.QuoteSingle, got.Quote)
}

func TestRejectsLine(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		src  string
		err  error
	}{
		{name: "words", src: "not-an-assignment\n", err: ErrLine},
		{name: "unclosed", src: "KEY=\"foo\n", err: ErrLine},
		{name: "unclosed single", src: "KEY='foo\n", err: ErrLine},
		{name: "junk", src: "KEY=\"foo\" bar\n", err: ErrLine},
		{name: "digit", src: "1KEY=value\n", err: ErrLine},
		{name: "empty key", src: "=value\n", err: ErrKey},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := Take(strings.NewReader(tc.src))
			require.ErrorIs(t, err, tc.err)
			if tc.err == ErrKey {
				require.ErrorIs(t, err, errs.ErrEmpty)
				return
			}
			require.ErrorIs(t, err, errs.ErrSyntax)
		})
	}
}

func TestMakeRejectsQuote(t *testing.T) {
	t.Parallel()
	bare := New().Assign("M", "a\nb")
	require.ErrorIs(t, bare.Make(io.Discard), ErrEncode)
	single := New().Assign("M", "it's").Single()
	require.ErrorIs(t, single.Make(io.Discard), ErrEncode)
	broken := New().Assign("M", "a\nb").Single()
	require.ErrorIs(t, broken.Make(io.Discard), ErrEncode)
	require.ErrorIs(t, bare.Make(io.Discard), errs.ErrInvalid)
	unset := New().Assign("A", "1")
	unset.Assignments[0].Quote = 0
	require.ErrorIs(t, unset.Make(io.Discard), syntax.ErrQuote)
	require.ErrorIs(t, unset.Make(io.Discard), errs.ErrUnknown)
}

func TestNil(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrNilReader)
	require.ErrorIs(t, err, errs.ErrInvalid)
	var file *File
	require.ErrorIs(t, file.Make(io.Discard), ErrNilFile)
	require.ErrorIs(t, New().Make(nil), ErrNilWriter)
}

func TestReadError(t *testing.T) {
	t.Parallel()
	_, err := Take(failReader{})
	require.ErrorIs(t, err, errs.ErrUnavailable)
}

func TestMakeWritesThrough(t *testing.T) {
	t.Parallel()
	var got countWriter
	require.NoError(t, New().Assign("KEY", "value").Make(&got))
	require.Greater(t, got.writes, 1)
	require.Equal(t, "KEY=value\n", got.String())
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	file, printed := takeMake(t, "\n\n")
	require.Empty(t, file.Assignments)
	require.Empty(t, printed)
}

type view struct {
	key     string
	value   string
	comment string
	quote   syntax.Quote
	export  bool
}

func views(file *File) []view {
	out := make([]view, len(file.Assignments))
	for i, assignment := range file.Assignments {
		out[i] = view{
			key:     assignment.Key,
			value:   assignment.Value,
			comment: assignment.Comment,
			quote:   assignment.Quote,
			export:  assignment.Export,
		}
	}
	return out
}

func takeMake(t *testing.T, src string) (*File, string) {
	t.Helper()
	file, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	return file, buf.String()
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) {
	return 0, errs.ErrUnavailable
}

type countWriter struct {
	bytes.Buffer
	writes int
}

func (c *countWriter) Write(p []byte) (int, error) {
	c.writes++
	return c.Buffer.Write(p)
}

func (c *countWriter) WriteString(s string) (int, error) {
	c.writes++
	return c.Buffer.WriteString(s)
}
