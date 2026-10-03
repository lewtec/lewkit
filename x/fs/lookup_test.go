package fs

import (
	iofs "io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memEntry struct {
	name string
	dir  bool
	kids []memEntry
}

func (m memEntry) Name() string { return m.name }
func (m memEntry) IsDir() bool  { return m.dir }

func readMem(m memEntry) ([]memEntry, error) { return m.kids, nil }

func TestLookup(t *testing.T) {
	t.Parallel()
	root := memEntry{name: ".", dir: true, kids: []memEntry{
		{name: "a", dir: true, kids: []memEntry{{name: "b.txt"}}},
		{name: "z.txt"},
	}}
	got, err := Lookup("open", "a/b.txt", root, readMem)
	require.NoError(t, err)
	assert.Equal(t, "b.txt", got.Name())

	got, err = Lookup("open", ".", root, readMem)
	require.NoError(t, err)
	assert.True(t, got.IsDir())

	_, err = Lookup("open", "missing", root, readMem)
	require.ErrorIs(t, err, iofs.ErrNotExist)

	_, err = Lookup("open", "../x", root, readMem)
	require.ErrorIs(t, err, iofs.ErrInvalid)

	_, err = Lookup("open", "z.txt/nope", root, readMem)
	require.ErrorIs(t, err, iofs.ErrInvalid)
}

func TestLookupFirstMatch(t *testing.T) {
	t.Parallel()
	root := memEntry{name: ".", dir: true, kids: []memEntry{
		{name: "a.txt"},
		{name: "a.txt", dir: true},
	}}
	got, err := Lookup("open", "a.txt", root, readMem)
	require.NoError(t, err)
	assert.False(t, got.IsDir())
}
