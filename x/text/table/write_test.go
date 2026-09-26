package table

import (
	"bytes"
	"errors"
	"iter"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	Note  string `json:"-"`
	err   error
	Inner item `json:"item"`
}

type item struct {
	Ok bool `json:"ok"`
}

var errBoom = errors.New("boom")

func samples() iter.Seq[sample] {
	return func(yield func(sample) bool) {
		yield(sample{Name: "a", Count: 2, Note: "hide", Inner: item{Ok: true}})
		yield(sample{Name: "b,c", Count: 0, err: errBoom})
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, Table, samples()))
	assert.Equal(t, strings.Join([]string{
		"name  count  item",
		"a     2      {\"ok\":true}",
		"b,c   0      {\"ok\":false}",
		"",
	}, "\n"), buf.String())
}

func TestWriteJSONL(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, JSONL, samples()))
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 2)
	assert.JSONEq(t, `{"name":"a","count":2,"item":{"ok":true}}`, lines[0])
	assert.JSONEq(t, `{"name":"b,c","count":0,"item":{"ok":false}}`, lines[1])
	assert.NotContains(t, buf.String(), "hide")
	assert.NotContains(t, buf.String(), "boom")
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, CSV, samples()))
	assert.Equal(t, "name,count,item\na,2,\"{\"\"ok\"\":true}\"\n\"b,c\",0,\"{\"\"ok\"\":false}\"\n", buf.String())
}

func TestWriteEmptyStillHasHeader(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, Table, iter.Seq[sample](nil)))
	assert.Equal(t, "name  count  item\n", buf.String())
}

func TestWriteScalar(t *testing.T) {
	var buf bytes.Buffer
	seq := func(yield func(string) bool) {
		yield("one")
		yield("two")
	}
	require.NoError(t, Write(&buf, CSV, seq))
	assert.Equal(t, "value\none\ntwo\n", buf.String())
}

func TestWriteRejectsFormat(t *testing.T) {
	err := Write(ioDiscard{}, Format(0), samples())
	assert.ErrorIs(t, err, ErrFormat)
}

type ordered struct {
	Name string `json:"name" table:",order=1"`
	ID   string `json:"id" table:"ident,order=0"`
}

func TestWriteColumnOrder(t *testing.T) {
	seq := func(yield func(ordered) bool) {
		yield(ordered{Name: "a", ID: "z"})
	}
	var tableBuf bytes.Buffer
	require.NoError(t, Write(&tableBuf, Table, seq))
	assert.Equal(t, "ident  name\nz      a\n", tableBuf.String())

	var jsonBuf bytes.Buffer
	require.NoError(t, Write(&jsonBuf, JSONL, seq))
	assert.Equal(t, "{\"ident\":\"z\",\"name\":\"a\"}\n", jsonBuf.String())
}

type shown struct {
	When label `json:"when"`
}

type label struct {
	Raw string
}

func (l label) String() string { return "shown:" + l.Raw }

func (l label) MarshalJSON() ([]byte, error) { return []byte(`"json"`), nil }

func TestWriteStringerKeepsJSON(t *testing.T) {
	seq := func(yield func(shown) bool) {
		yield(shown{When: label{Raw: "x"}})
	}
	var tableBuf bytes.Buffer
	require.NoError(t, Write(&tableBuf, Table, seq))
	assert.Equal(t, "when\nshown:x\n", tableBuf.String())

	var jsonBuf bytes.Buffer
	require.NoError(t, Write(&jsonBuf, JSONL, seq))
	assert.Equal(t, "{\"when\":\"json\"}\n", jsonBuf.String())
}

func TestWriteExplicitColumns(t *testing.T) {
	var buf bytes.Buffer
	err := Write(&buf, JSONL, samples(),
		Column[sample]{Name: "n", Text: func(s sample) string { return strings.ToUpper(s.Name) }, JSON: func(s sample) any { return s.Count }},
	)
	require.NoError(t, err)
	assert.Equal(t, "{\"n\":2}\n{\"n\":0}\n", buf.String())

	buf.Reset()
	err = Write(&buf, Table, samples(),
		Column[sample]{Name: "n", Text: func(s sample) string { return strings.ToUpper(s.Name) }},
	)
	require.NoError(t, err)
	assert.Equal(t, "n\nA\nB,C\n", buf.String())
}

func TestWriteRejectsBadColumn(t *testing.T) {
	err := Write(ioDiscard{}, Table, samples(), Column[sample]{Name: "n"})
	assert.ErrorIs(t, err, ErrColumn)
}

func TestWriteRejectsBadOrder(t *testing.T) {
	type bad struct {
		Name string `table:",order=nope"`
	}
	err := Write(ioDiscard{}, Table, func(func(bad) bool) {})
	assert.ErrorIs(t, err, ErrColumn)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
