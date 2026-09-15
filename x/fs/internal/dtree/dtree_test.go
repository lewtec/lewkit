package dtree

import (
	"io"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddLookup(t *testing.T) {
	root := NewDir[int](".")
	require.NoError(t, root.Add("a/b.txt", 1, false))
	require.NoError(t, root.Add("a/c", 0, true))
	require.NoError(t, root.Add("z.txt", 2, false))

	n, err := LookupPath(root, "a/b.txt")
	require.NoError(t, err)
	assert.Equal(t, 1, n.Val)
	assert.False(t, n.Dir)

	n, err = LookupPath(root, "a")
	require.NoError(t, err)
	assert.True(t, n.Dir)

	_, err = LookupPath(root, "missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = LookupPath(root, "../x")
	require.ErrorIs(t, err, fs.ErrInvalid)

	require.ErrorIs(t, root.Add("a/b.txt", 3, false), fs.ErrExist)
	require.NoError(t, root.Add("a", 4, true))
	n, err = LookupPath(root, "a")
	require.NoError(t, err)
	assert.Equal(t, 4, n.Val)
}

func TestDirFileReadDir(t *testing.T) {
	root := NewDir[int](".")
	require.NoError(t, root.Add("z.txt", 1, false))
	require.NoError(t, root.Add("a", 0, true))
	info := func(n *Node[int]) fs.FileInfo {
		mode := fs.FileMode(0o444)
		if n.Dir {
			mode = fs.ModeDir | 0o555
		}
		return Info(n.Name, 0, mode, time.Time{})
	}
	d := &DirFile{Info: Info(".", 0, fs.ModeDir|0o555, time.Time{}), Ents: root.Entries(info)}
	ents, err := d.ReadDir(-1)
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())
	_, err = d.ReadDir(1)
	require.ErrorIs(t, err, io.EOF)
	_, err = d.Read(nil)
	require.ErrorIs(t, err, fs.ErrInvalid)
}
