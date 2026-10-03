package udf

import (
	"bytes"
	"errors"
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

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := test.ReaderOnlyFS("vol.iso", strings.NewReader("x"))
	_, err := path.OpenFS(t.Context(), path.New("vol.iso"), fsys, Open)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), test.OnlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestInvalidVolume(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), bytes.NewReader(make([]byte, 4096)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
}

func TestTree(t *testing.T) {
	t.Parallel()
	root := memDir(".",
		memDir("a", memFileEnt("b.txt"), memDir("c")),
		memFileEnt("z.txt"),
	)
	fsys := &FS{root: root}

	ents, err := fsys.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())
	assert.False(t, ents[1].IsDir())

	ents, err = fsys.ReadDir("a")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "b.txt", ents[0].Name())
	assert.Equal(t, "c", ents[1].Name())
	assert.True(t, ents[1].IsDir())

	st, err := fsys.Stat("a/c")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.Open("../x")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadFile("a")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadDir("z.txt")
	require.ErrorIs(t, err, fs.ErrInvalid)

	f, err := fsys.Open("z.txt")
	require.NoError(t, err)
	require.NoError(t, f.Close())
}

func TestPathReadDir(t *testing.T) {
	t.Parallel()
	root := memDir(".", memDir("sources", memFileEnt("install.wim")))
	fsys := &FS{root: root}
	ents, err := path.New("sources").ReadDir(fsys)
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "install.wim", ents[0].Name())
}

func TestLookupFirstName(t *testing.T) {
	t.Parallel()
	root := memDir(".", memFileEnt("a.txt"), memFileEnt("a.txt"))
	root.kids[1].body = []byte("second")
	fsys := &FS{root: root}
	b, err := fsys.ReadFile("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "first", string(b))
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
	body []byte
	kids []*mem
}

func memDir(name string, kids ...*mem) *mem {
	return &mem{name: name, dir: true, kids: kids}
}

func memFileEnt(name string) *mem {
	return &mem{name: name, body: []byte("first")}
}

func (m *mem) Name() string { return m.name }
func (m *mem) IsDir() bool  { return m.dir }

func (m *mem) info() fs.FileInfo {
	mode := fs.FileMode(0o444)
	if m.dir {
		mode = fs.ModeDir | 0o555
	}
	return lewfs.FileInfo(m.name, int64(len(m.body)), mode, time.Time{})
}

func (m *mem) open() (fs.File, error) {
	if m.dir {
		return nil, &fs.PathError{Op: "open", Path: m.name, Err: fs.ErrInvalid}
	}
	return &memFile{info: m.info(), r: bytes.NewReader(m.body)}, nil
}

func (m *mem) children() ([]entry, error) {
	out := make([]entry, len(m.kids))
	for i, k := range m.kids {
		out[i] = k
	}
	return out, nil
}

type memFile struct {
	info fs.FileInfo
	r    *bytes.Reader
}

func (f *memFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *memFile) Read(p []byte) (int, error) { return f.r.Read(p) }
func (f *memFile) Close() error               { return nil }
