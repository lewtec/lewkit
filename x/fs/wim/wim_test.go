//go:build linux || windows

package wim

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"time"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	r := test.OnlyReader{strings.NewReader("x")}
	_, err := Open(t.Context(), r, 1)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)

	_, err = Images(t.Context(), r)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}

func TestInvalidWim(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), bytes.NewReader(make([]byte, 256)), 1)
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
	assert.False(t, errors.Is(err, ErrInvalidImage))
}

func TestTree(t *testing.T) {
	t.Parallel()
	root := memDir(".",
		memDir("Windows",
			memDir("Fonts", memFileEnt("arial.ttf")),
			memDir("System32"),
		),
	)
	fsys := &FS{root: root}

	ents, err := fsys.ReadDir("Windows")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "Fonts", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "System32", ents[1].Name())

	st, err := fsys.Stat("Windows/Fonts/arial.ttf")
	require.NoError(t, err)
	assert.False(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.ReadFile("Windows")
	require.ErrorIs(t, err, fs.ErrInvalid)

	require.NoError(t, fsys.Close())
	_, err = fsys.Open("Windows")
	require.ErrorIs(t, err, fs.ErrClosed)
}

func TestPathReadDir(t *testing.T) {
	t.Parallel()
	root := memDir(".", memDir("Windows", memDir("Fonts", memFileEnt("arial.ttf"))))
	fsys := &FS{root: root}
	ents, err := path.New("Windows", "Fonts").ReadDir(fsys)
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "arial.ttf", ents[0].Name())
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	fsys := &FS{root: memDir(".", memFileEnt("a.txt"))}
	err := path.New("a.txt").WriteFile(fsys, []byte("x"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

type mem struct {
	name string
	dir  bool
	kids []*mem
}

func memDir(name string, kids ...*mem) *mem {
	return &mem{name: name, dir: true, kids: kids}
}

func memFileEnt(name string) *mem { return &mem{name: name} }

func (m *mem) Name() string { return m.name }
func (m *mem) IsDir() bool  { return m.dir }

func (m *mem) info() fs.FileInfo {
	mode := fs.FileMode(0o444)
	if m.dir {
		mode = fs.ModeDir | 0o555
	}
	return lewfs.FileInfo(m.name, 0, mode, time.Time{})
}

func (m *mem) open() (fs.File, error) {
	if m.dir {
		return nil, &fs.PathError{Op: "open", Path: m.name, Err: fs.ErrInvalid}
	}
	return nopFile{info: m.info()}, nil
}

func (m *mem) children() ([]entry, error) {
	out := make([]entry, len(m.kids))
	for i, k := range m.kids {
		out[i] = k
	}
	return out, nil
}

type nopFile struct{ info fs.FileInfo }

func (f nopFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (nopFile) Read([]byte) (int, error)     { return 0, io.EOF }
func (nopFile) Close() error                 { return nil }
