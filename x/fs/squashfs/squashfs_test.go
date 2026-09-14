package squashfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestInvalidImage(t *testing.T) {
	t.Parallel()
	_, err := Open(bytes.NewReader(make([]byte, 4096)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
}

func TestTree(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/tiny.sfs")
	require.NoError(t, err)
	fsys, err := Open(bytes.NewReader(raw))
	require.NoError(t, err)

	ents, err := fsys.ReadDir(".")
	require.NoError(t, err)
	require.Len(t, ents, 2)
	assert.Equal(t, "a", ents[0].Name())
	assert.True(t, ents[0].IsDir())
	assert.Equal(t, "z.txt", ents[1].Name())
	assert.False(t, ents[1].IsDir())

	ents, err = fsys.ReadDir("a")
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "b.txt", ents[0].Name())

	b, err := fsys.ReadFile("a/b.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello\n", string(b))

	st, err := fsys.Stat("a")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.ReadFile("a")
	require.Error(t, err)
}

func TestPathReadDir(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/tiny.sfs")
	require.NoError(t, err)
	fsys, err := Open(bytes.NewReader(raw))
	require.NoError(t, err)
	ents, err := path.New("a").ReadDir(fsys)
	require.NoError(t, err)
	require.Len(t, ents, 1)
	assert.Equal(t, "b.txt", ents[0].Name())
}

func TestFS(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/tiny.sfs")
	require.NoError(t, err)
	fsys, err := Open(bytes.NewReader(raw))
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/tiny.sfs")
	require.NoError(t, err)
	fsys, err := Open(bytes.NewReader(raw))
	require.NoError(t, err)
	err = path.New("z.txt").WriteFile(fsys, []byte("x"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := &readerOnlyFS{name: "root.sfs", r: strings.NewReader("x")}
	_, err := path.OpenFS(path.New("root.sfs"), fsys, Open)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}

type readerOnlyFS struct {
	name string
	r    io.Reader
}

func (s *readerOnlyFS) Open(name string) (fs.File, error) {
	if name != s.name {
		return nil, fs.ErrNotExist
	}
	return readerOnlyFile{Reader: s.r}, nil
}

type readerOnlyFile struct{ io.Reader }

func (readerOnlyFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (readerOnlyFile) Close() error               { return nil }
