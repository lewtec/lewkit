package android

import (
	"io"
	"io/fs"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestDocumentTree(t *testing.T) {
	t.Parallel()
	const root = "content://com.example/tree/root"
	const readme = "content://com.example/tree/root/document/readme"
	const pics = "content://com.example/tree/root/document/pics"
	const photo = "content://com.example/tree/root/document/photo"
	docs := &memDocs{
		info: map[string]Doc{
			root: {Name: "root", Dir: true, URI: root},
		},
		kids: map[string][]Doc{
			root: {
				{Name: "readme.txt", Size: 5, URI: readme},
				{Name: "pics", Dir: true, URI: pics},
			},
			pics: {
				{Name: "a.jpg", Size: 3, URI: photo},
			},
		},
		body: map[string]string{
			readme: "hello",
			photo:  "jpg",
		},
	}
	fsys, err := FS([]string{root}, docs)
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "readme.txt", "pics", "pics/a.jpg"))
	body, err := fs.ReadFile(fsys, "readme.txt")
	require.NoError(t, err)
	require.Equal(t, "hello", string(body))
	_, err = fsys.Open("..")
	require.ErrorIs(t, err, fs.ErrInvalid)
	n := docs.lists
	require.NotZero(t, n)
	_, err = fs.ReadDir(fsys, ".")
	require.NoError(t, err)
	_, err = fs.ReadDir(fsys, "pics")
	require.NoError(t, err)
	require.Equal(t, n, docs.lists)
}

func TestPreviewIsNotTheFile(t *testing.T) {
	t.Parallel()
	const photo = "content://photo"
	docs := &thumbDocs{
		memDocs: memDocs{
			info: map[string]Doc{photo: {Name: "a.jpg", Size: 26 << 20, URI: photo}},
			body: map[string]string{photo: "original"},
		},
		thumb: "small",
	}
	fsys, err := FS([]string{photo}, docs)
	require.NoError(t, err)
	body, err := fs.ReadFile(fsys, "a.jpg")
	require.NoError(t, err)
	require.Equal(t, "original", string(body))
	prev, err := fsys.(interface{ Preview(string) (fs.File, error) }).Preview("a.jpg")
	require.NoError(t, err)
	defer prev.Close()
	got, err := io.ReadAll(prev)
	require.NoError(t, err)
	require.Equal(t, "small", string(got))
}

func TestDocumentFile(t *testing.T) {
	t.Parallel()
	const uri = "content://com.example/document/notes"
	fsys, err := FS([]string{uri}, &memDocs{
		info: map[string]Doc{uri: {Name: "notes.txt", Size: 2, URI: uri}},
		body: map[string]string{uri: "hi"},
	})
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "notes.txt"))
	body, err := fs.ReadFile(fsys, "notes.txt")
	require.NoError(t, err)
	require.Equal(t, "hi", string(body))
}

func TestDocumentNames(t *testing.T) {
	t.Parallel()
	const root = "content://tree"
	docs := &memDocs{
		info: map[string]Doc{root: {Dir: true, URI: root}},
		kids: map[string][]Doc{
			root: {
				{Name: "a.txt", Size: 1, URI: "content://1"},
				{Name: "a.txt", Size: 1, URI: "content://2"},
				{Name: "a/b", Size: 1, URI: "content://3"},
				{Name: "..", Dir: true, URI: "content://4"},
			},
		},
		body: map[string]string{
			"content://1": "1",
			"content://2": "2",
			"content://3": "3",
		},
		kidsEmpty: map[string]bool{"content://4": true},
	}
	fsys, err := FS([]string{root}, docs)
	require.NoError(t, err)
	entries, err := fs.ReadDir(fsys, ".")
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.Equal(t, []string{"_", "a (2).txt", "a.txt", "a_b"}, names)
	body, err := fs.ReadFile(fsys, "a (2).txt")
	require.NoError(t, err)
	require.Equal(t, "2", string(body))
	sub, err := fs.ReadDir(fsys, "_")
	require.NoError(t, err)
	require.Empty(t, sub)
}

func TestSeveralDocuments(t *testing.T) {
	t.Parallel()
	const a = "content://a"
	const b = "content://b"
	fsys, err := FS([]string{a, b}, &memDocs{
		info: map[string]Doc{
			a: {Name: "note.txt", Size: 1, URI: a},
			b: {Name: "note.txt", Size: 1, URI: b},
		},
		body: map[string]string{a: "a", b: "b"},
	})
	require.NoError(t, err)
	require.NoError(t, fstest.TestFS(fsys, "note.txt", "note (2).txt"))
	body, err := fs.ReadFile(fsys, "note (2).txt")
	require.NoError(t, err)
	require.Equal(t, "b", string(body))
}

func TestCatalog(t *testing.T) {
	t.Parallel()
	docs := []Doc{{
		Name: "a\\b\n\r\t",
		Size: 12,
		Dir:  true,
		URI:  "content://x y",
	}}
	got, err := parseCatalog(catalog(docs...))
	require.NoError(t, err)
	require.Equal(t, docs, got)

	got, err = parseCatalog(catalog(docs...) + "\n")
	require.NoError(t, err)
	require.Equal(t, docs, got)

	got, err = parseCatalog("")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = parseCatalog("a\t1")
	require.ErrorIs(t, err, errCatalog)
	_, err = parseCatalog("a\tx\t0\turi")
	require.ErrorIs(t, err, errCatalog)
	_, err = parseCatalog("a\t1\t2\turi")
	require.ErrorIs(t, err, errCatalog)
	_, err = parseCatalog("a\\\t1\t0\turi")
	require.ErrorIs(t, err, errCatalog)
}

func catalog(docs ...Doc) string {
	var b strings.Builder
	for i, d := range docs {
		if i > 0 {
			b.WriteByte('\n')
		}
		dir := "0"
		if d.Dir {
			dir = "1"
		}
		b.WriteString(escapeField(d.Name))
		b.WriteByte('\t')
		b.WriteString(strconv.FormatInt(d.Size, 10))
		b.WriteByte('\t')
		b.WriteString(dir)
		b.WriteByte('\t')
		b.WriteString(escapeField(d.URI))
	}
	return b.String()
}

func escapeField(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

type memDocs struct {
	info      map[string]Doc
	kids      map[string][]Doc
	kidsEmpty map[string]bool
	body      map[string]string
	lists     int
}

func (m *memDocs) Info(uri string) (Doc, error) {
	doc, ok := m.info[uri]
	if !ok {
		return Doc{}, fs.ErrNotExist
	}
	return doc, nil
}

func (m *memDocs) List(uri string) ([]Doc, error) {
	m.lists++
	if m.kidsEmpty[uri] {
		return nil, nil
	}
	kids, ok := m.kids[uri]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return append([]Doc(nil), kids...), nil
}

func (m *memDocs) Open(uri string) (fs.File, error) {
	body, ok := m.body[uri]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return memFile{r: strings.NewReader(body)}, nil
}

type thumbDocs struct {
	memDocs
	thumb string
}

func (d *thumbDocs) Thumb(uri string) (fs.File, error) {
	if uri == "" {
		return nil, fs.ErrNotExist
	}
	return memFile{r: strings.NewReader(d.thumb)}, nil
}

type memFile struct{ r *strings.Reader }

func (m memFile) Stat() (fs.FileInfo, error) { return nil, fs.ErrInvalid }
func (m memFile) Read(p []byte) (int, error) { return m.r.Read(p) }
func (m memFile) Close() error               { return nil }
