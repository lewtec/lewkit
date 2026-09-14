package tar

import (
	stdtar "archive/tar"
	"bytes"
	"errors"
	"io"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	lewfs "github.com/lewtec/lewkit/x/fs"
	"github.com/lewtec/lewkit/x/path"
	"github.com/lewtec/lewkit/x/test"

	stdgzip "compress/gzip"
	"github.com/andybalholm/brotli"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type onlyReader struct{ io.Reader }

func packTar(t *testing.T, files map[string][]byte) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := stdtar.NewWriter(&buf)
	for name, data := range files {
		hdr := &stdtar.Header{Name: name, Mode: 0o644, Size: int64(len(data))}
		require.NoError(t, w.WriteHeader(hdr))
		_, err := w.Write(data)
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return bytes.NewReader(buf.Bytes())
}

func packTarWrapped(t *testing.T, files map[string][]byte, wrap func(io.Writer) io.WriteCloser) *bytes.Reader {
	t.Helper()
	raw := packTar(t, files)
	var buf bytes.Buffer
	w := wrap(&buf)
	_, err := io.Copy(w, raw)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return bytes.NewReader(buf.Bytes())
}

func TestFiles(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	var names []string
	for f, err := range Files(r) {
		require.NoError(t, err)
		names = append(names, f.Name.String())
	}
	assert.ElementsMatch(t, []string{"a/b.txt", "z.txt"}, names)

	got := test.Collect(t, path.New(".").Select(lewfs.Names(Files(r)), "**/*.txt"))
	assert.ElementsMatch(t, []string{"a/b.txt", "z.txt"}, namesOf(got))
}

func TestFilesStream(t *testing.T) {
	t.Parallel()
	raw := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	var saw string
	for f, err := range Files(onlyReader{bytes.NewReader(data)}) {
		require.NoError(t, err)
		if f.Name.String() != "z.txt" {
			continue
		}
		require.NotNil(t, f.Open)
		rf, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rf)
		require.NoError(t, err)
		require.NoError(t, rf.Close())
		saw = string(b)
	}
	assert.Equal(t, "zee", saw)
}

func namesOf(ps []path.Path) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}

func TestNeedReadAt(t *testing.T) {
	t.Parallel()
	_, err := Open(onlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestInvalidTar(t *testing.T) {
	t.Parallel()
	_, err := Open(bytes.NewReader(bytes.Repeat([]byte("x"), 512)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
}

func TestTree(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	fsys, err := Open(r)
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
	assert.Equal(t, "hello", string(b))

	st, err := fsys.Stat("a")
	require.NoError(t, err)
	assert.True(t, st.IsDir())

	_, err = fsys.Open("missing")
	require.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fsys.Open("../x")
	require.ErrorIs(t, err, fs.ErrInvalid)

	_, err = fsys.ReadFile("a")
	require.ErrorIs(t, err, fs.ErrInvalid)
}

func TestReadAt(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{"a.bin": []byte("hello world")})
	fsys, err := Open(r)
	require.NoError(t, err)
	f, err := fsys.Open("a.bin")
	require.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	ra, ok := f.(io.ReaderAt)
	require.True(t, ok)
	buf := make([]byte, 5)
	n, err := ra.ReadAt(buf, 6)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "world", string(buf))
}

func TestSkipPayload(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"big.bin": bytes.Repeat([]byte("x"), 1<<20),
		"ok.txt":  []byte("yes"),
	})
	fsys, err := Open(r)
	require.NoError(t, err)
	b, err := fsys.ReadFile("ok.txt")
	require.NoError(t, err)
	assert.Equal(t, "yes", string(b))
}

func TestOpenGzipMagic(t *testing.T) {
	t.Parallel()
	r := packTarWrapped(t, map[string][]byte{"a/b.txt": []byte("hello")}, func(w io.Writer) io.WriteCloser {
		return stdgzip.NewWriter(w)
	})
	fsys, err := Open(r)
	require.NoError(t, err)
	b, err := fsys.ReadFile("a/b.txt")
	require.NoError(t, err)
	assert.Equal(t, "hello", string(b))
}

func TestOpenBrotliByName(t *testing.T) {
	t.Parallel()
	raw := packTarWrapped(t, map[string][]byte{"a.txt": []byte("x")}, func(w io.Writer) io.WriteCloser {
		return brotli.NewWriter(w)
	})
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	host := fstest.MapFS{"src.tar.br": {Data: data}}
	fsys, err := path.OpenFS(path.New("src.tar.br"), host, Open)
	require.NoError(t, err)
	b, err := path.New("a.txt").ReadFile(fsys)
	require.NoError(t, err)
	assert.Equal(t, []byte("x"), b)
}

func TestOpenBrotliNeedsName(t *testing.T) {
	t.Parallel()
	r := packTarWrapped(t, map[string][]byte{"a.txt": []byte("x")}, func(w io.Writer) io.WriteCloser {
		return brotli.NewWriter(w)
	})
	_, err := Open(r)
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
}

func TestOpenGzipWithoutReadAt(t *testing.T) {
	t.Parallel()
	r := packTarWrapped(t, map[string][]byte{"a.txt": []byte("x")}, func(w io.Writer) io.WriteCloser {
		return stdgzip.NewWriter(w)
	})
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	fsys, err := Open(onlyReader{bytes.NewReader(data)})
	require.NoError(t, err)
	b, err := fsys.ReadFile("a.txt")
	require.NoError(t, err)
	assert.Equal(t, "x", string(b))
}

func TestOpenFS(t *testing.T) {
	t.Parallel()
	raw := packTar(t, map[string][]byte{"README": []byte("hi")})
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	host := fstest.MapFS{"src.tar": {Data: data}}
	tf, err := path.OpenFS(path.New("src.tar"), host, Open)
	require.NoError(t, err)
	b, err := path.New("README").ReadFile(tf)
	require.NoError(t, err)
	assert.Equal(t, []byte("hi"), b)
}

func TestFS(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	fsys, err := Open(r)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{"a.txt": []byte("x")})
	fsys, err := Open(r)
	require.NoError(t, err)
	err = path.New("a.txt").WriteFile(fsys, []byte("y"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := &readerOnlyFS{name: "a.tar", r: strings.NewReader("x")}
	_, err := path.OpenFS(path.New("a.tar"), fsys, Open)
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
