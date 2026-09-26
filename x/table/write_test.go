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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
