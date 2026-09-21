package tar

import (
	stdtar "archive/tar"
	"bytes"
	"context"
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

func TestFilesCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var got error
	for _, err := range Files(ctx, packTar(t, map[string][]byte{"a.txt": []byte("x")})) {
		got = err
	}
	require.ErrorIs(t, got, context.Canceled)
}

func TestOpenCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := Open(ctx, packTar(t, map[string][]byte{"a.txt": []byte("x")}))
	require.ErrorIs(t, err, context.Canceled)
}

func TestFiles(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	var names []string
	for f, err := range Files(t.Context(), r) {
		require.NoError(t, err)
		names = append(names, f.Name.String())
	}
	assert.ElementsMatch(t, []string{"a/b.txt", "z.txt"}, names)

	got := test.Collect(t, path.New(".").Select(lewfs.Names(Files(t.Context(), r)), "**/*.txt"))
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
	for f, err := range Files(t.Context(), test.OnlyReader{bytes.NewReader(data)}) {
		require.NoError(t, err)
		if f.Name.String() != "z.txt" {
			continue
		}
		require.NotNil(t, f.Reader)
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
	_, err := Open(t.Context(), test.OnlyReader{strings.NewReader("x")})
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
	pe, ok := errors.AsType[*fs.PathError](err)
	require.True(t, ok)
	assert.Equal(t, "open", pe.Op)
}

func TestInvalidTar(t *testing.T) {
	t.Parallel()
	_, err := Open(t.Context(), bytes.NewReader(bytes.Repeat([]byte("x"), 512)))
	require.Error(t, err)
	assert.False(t, errors.Is(err, lewfs.ErrNeedReadAt))
}

func TestTree(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	test.ArchiveTree(t, fsys)
	_, err = fsys.ReadFile("a")
	require.ErrorIs(t, err, fs.ErrInvalid)
}

func TestReadAt(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{"a.bin": []byte("hello world")})
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	test.ArchiveReadAt(t, fsys)
}

func TestSkipPayload(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{
		"big.bin": bytes.Repeat([]byte("x"), 1<<20),
		"ok.txt":  []byte("yes"),
	})
	fsys, err := Open(t.Context(), r)
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
	fsys, err := Open(t.Context(), r)
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
	fsys, err := path.OpenFS(t.Context(), path.New("src.tar.br"), host, Open)
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
	_, err := Open(t.Context(), r)
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
	fsys, err := Open(t.Context(), test.OnlyReader{bytes.NewReader(data)})
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
	tf, err := path.OpenFS(t.Context(), path.New("src.tar"), host, Open)
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
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "a/b.txt", "z.txt"))
}

func TestCopyExtract(t *testing.T) {
	t.Parallel()
	raw := packTar(t, map[string][]byte{
		"a/b.txt": []byte("hello"),
		"z.txt":   []byte("zee"),
	})
	data, err := io.ReadAll(raw)
	require.NoError(t, err)
	dest, err := path.Open(t.TempDir())
	require.NoError(t, err)
	test.CloseOnCleanup(t, dest)
	require.NoError(t, lewfs.Copy(t.Context(), dest, Files(t.Context(), test.OnlyReader{bytes.NewReader(data)})))
	b, err := path.New("z.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("zee"), b)
	b, err = path.New("a", "b.txt").ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, []byte("hello"), b)
}

func TestWriteReadOnly(t *testing.T) {
	t.Parallel()
	r := packTar(t, map[string][]byte{"a.txt": []byte("x")})
	fsys, err := Open(t.Context(), r)
	require.NoError(t, err)
	err = path.New("a.txt").WriteFile(fsys, []byte("y"), 0o644)
	require.ErrorIs(t, err, path.ErrReadOnly)
}

func TestOpenFSNeedReadAt(t *testing.T) {
	t.Parallel()
	fsys := test.ReaderOnlyFS("a.tar", strings.NewReader("x"))
	_, err := path.OpenFS(t.Context(), path.New("a.tar"), fsys, Open)
	require.ErrorIs(t, err, lewfs.ErrNeedReadAt)
}
