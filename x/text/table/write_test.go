package table

import (
	"bytes"
	"errors"
	"iter"
	"strings"
	"testing"
	"time"

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
		if !yield(sample{Name: "a", Count: 2, Note: "hide", Inner: item{Ok: true}}) {
			return
		}
		yield(sample{Name: "b,c", Count: 0, err: errBoom})
	}
}

func TestWriteTable(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, Table, samples(), nil))
	assert.Equal(t, strings.Join([]string{
		"name  count  item",
		"a     2      {\"ok\":true}",
		"b,c   0      {\"ok\":false}",
		"",
	}, "\n"), buf.String())
}

func TestWriteJSONL(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, JSONL, samples(), nil))
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	require.Len(t, lines, 2)
	assert.JSONEq(t, `{"name":"a","count":2,"item":{"ok":true}}`, lines[0])
	assert.JSONEq(t, `{"name":"b,c","count":0,"item":{"ok":false}}`, lines[1])
	assert.NotContains(t, buf.String(), "hide")
	assert.NotContains(t, buf.String(), "boom")
}

func TestWriteCSV(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, CSV, samples(), nil))
	assert.Equal(t, "name,count,item\na,2,\"{\"\"ok\"\":true}\"\n\"b,c\",0,\"{\"\"ok\"\":false}\"\n", buf.String())
}

func TestWriteEmptyStillHasHeader(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, Table, iter.Seq[sample](nil), nil))
	assert.Equal(t, "name  count  item\n", buf.String())
}

func TestWriteScalar(t *testing.T) {
	var buf bytes.Buffer
	seq := func(yield func(string) bool) {
		yield("one")
		yield("two")
	}
	require.NoError(t, Write(&buf, CSV, seq, nil))
	assert.Equal(t, "value\none\ntwo\n", buf.String())
}

func TestWriteRejectsFormat(t *testing.T) {
	err := Write(ioDiscard{}, Format(0), samples(), nil)
	assert.ErrorIs(t, err, ErrFormat)
}

type listed struct {
	Name string
	When time.Time
}

func (listed) Columns() []Column[listed] {
	return []Column[listed]{
		{Name: "name", Value: func(row listed) any { return row.Name }},
		{Name: "when", Value: func(row listed) any { return row.When }, Format: time.DateOnly},
	}
}

func TestWriteUsesColumnsMethod(t *testing.T) {
	when := time.Date(2026, 9, 26, 15, 4, 0, 0, time.UTC)
	seq := func(yield func(listed) bool) {
		yield(listed{Name: "a", When: when})
	}
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, JSONL, seq, nil))
	assert.Equal(t, "{\"name\":\"a\",\"when\":\"2026-09-26\"}\n", buf.String())
}

func TestSelectReordersAndFormats(t *testing.T) {
	cols, err := Resolve[listed](nil)
	require.NoError(t, err)
	cols, err = Select(cols, "when=%s,name=%q")
	require.NoError(t, err)
	when := time.Date(2026, 9, 26, 15, 4, 0, 0, time.UTC)
	seq := func(yield func(listed) bool) {
		yield(listed{Name: "a", When: when})
	}
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, JSONL, seq, Fields[listed](cols)))
	assert.Equal(t, "{\"when\":\"2026-09-26 15:04:00 +0000 UTC\",\"name\":\"\\\"a\\\"\"}\n", buf.String())
}

func TestSelectRejectsUnknown(t *testing.T) {
	cols, err := Resolve[listed](nil)
	require.NoError(t, err)
	_, err = Select(cols, "missing")
	assert.ErrorIs(t, err, ErrColumn)
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
	require.NoError(t, Write(&tableBuf, Table, seq, nil))
	assert.Equal(t, "when\nshown:x\n", tableBuf.String())

	var jsonBuf bytes.Buffer
	require.NoError(t, Write(&jsonBuf, JSONL, seq, nil))
	assert.Equal(t, "{\"when\":\"json\"}\n", jsonBuf.String())
}

func TestWriteExplicitColumns(t *testing.T) {
	layout := Fields[sample]{{
		Name:  "n",
		Value: func(s sample) any { return s.Name },
	}}
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, Table, samples(), layout))
	assert.Equal(t, "n\na\nb,c\n", buf.String())
}

func TestWriteRejectsBadColumn(t *testing.T) {
	err := Write(ioDiscard{}, Table, samples(), Fields[sample]{{Name: "n"}})
	assert.ErrorIs(t, err, ErrColumn)
}

func TestWriteRejectsBadFormat(t *testing.T) {
	layout := Fields[sample]{{
		Name:   "n",
		Value:  func(s sample) any { return s.Name },
		Format: "nope",
	}}
	err := Write(ioDiscard{}, Table, samples(), layout)
	assert.ErrorIs(t, err, ErrColumn)
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
