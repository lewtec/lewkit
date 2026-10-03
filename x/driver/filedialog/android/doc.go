package android

import (
	"cmp"
	"errors"
	"io"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	errNoDocuments  = errors.New("no documents")
	errNoDocument   = errors.New("no document")
	errCatalog      = errors.New("document catalog")
	errDocumentInfo = errors.New("document info")
	errDocumentFd   = errors.New("document fd")
)

// Doc is one document in a provider tree.
type Doc struct {
	Name string
	Size int64
	Dir  bool
	URI  string
}

// Documents reads URIs from a document provider.
type Documents interface {
	Info(uri string) (Doc, error)
	List(uri string) ([]Doc, error)
	Open(uri string) (fs.File, error)
}

// FS is a filesystem over document URIs.
// One directory URI is the root. One file, or several URIs, share a root
// named by display name. Listings are cached for the life of the filesystem.
func FS(uris []string, docs Documents) (fs.FS, error) {
	if docs == nil {
		return nil, errNoDocuments
	}
	if len(uris) == 0 {
		return nil, errNoDocument
	}
	if len(uris) == 1 {
		info, err := docs.Info(uris[0])
		if err != nil {
			return nil, err
		}
		if info.URI == "" {
			info.URI = uris[0]
		}
		if info.Dir {
			return &docFS{docs: docs, root: info.URI, cache: map[string][]Doc{}}, nil
		}
		info.Name = cleanName(info.Name)
		return &docFS{docs: docs, tops: []Doc{info}, cache: map[string][]Doc{}}, nil
	}
	seen := map[string]struct{}{}
	tops := make([]Doc, 0, len(uris))
	for _, uri := range uris {
		info, err := docs.Info(uri)
		if err != nil {
			return nil, err
		}
		if info.URI == "" {
			info.URI = uri
		}
		info.Name = takeName(info.Name, seen)
		tops = append(tops, info)
	}
	slices.SortFunc(tops, func(a, b Doc) int { return cmp.Compare(a.Name, b.Name) })
	return &docFS{docs: docs, tops: tops, cache: map[string][]Doc{}}, nil
}

type docFS struct {
	docs  Documents
	root  string
	tops  []Doc
	mu    sync.Mutex
	cache map[string][]Doc
}

func (f *docFS) Open(name string) (fs.File, error) {
	doc, err := f.lookup(name)
	if err != nil {
		return nil, err
	}
	if name == "." && f.root == "" {
		return newDir(".", f.tops), nil
	}
	file, err := f.openDoc(doc)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return file, nil
}

// Preview is a small picture of a file. It is not the original bytes.
func (f *docFS) Preview(name string) (fs.File, error) {
	doc, err := f.lookup(name)
	if err != nil {
		return nil, err
	}
	thumbs, ok := f.docs.(thumbnailer)
	if !ok || doc.Dir || doc.URI == "" {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	file, err := thumbs.Thumb(doc.URI)
	if err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	}
	return file, nil
}

type thumbnailer interface {
	Thumb(uri string) (fs.File, error)
}

func (f *docFS) lookup(name string) (Doc, error) {
	if !fs.ValidPath(name) {
		return Doc{}, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	if name == "." {
		if f.root != "" {
			return Doc{Name: ".", Dir: true, URI: f.root}, nil
		}
		return Doc{Name: ".", Dir: true}, nil
	}
	if f.root != "" {
		return f.find(f.root, name, name)
	}
	head, rest, more := strings.Cut(name, "/")
	doc, ok := findDoc(f.tops, head)
	if !ok {
		return Doc{}, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if !more {
		return doc, nil
	}
	if !doc.Dir {
		return Doc{}, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return f.find(doc.URI, rest, name)
}

func (f *docFS) find(dirURI, rel, full string) (Doc, error) {
	cur := dirURI
	parts := strings.Split(rel, "/")
	var doc Doc
	for i, part := range parts {
		kids, err := f.children(cur)
		if err != nil {
			return Doc{}, &fs.PathError{Op: "open", Path: full, Err: err}
		}
		var ok bool
		doc, ok = findDoc(kids, part)
		if !ok {
			return Doc{}, &fs.PathError{Op: "open", Path: full, Err: fs.ErrNotExist}
		}
		if i == len(parts)-1 {
			return doc, nil
		}
		if !doc.Dir {
			return Doc{}, &fs.PathError{Op: "open", Path: full, Err: fs.ErrNotExist}
		}
		cur = doc.URI
	}
	return Doc{}, &fs.PathError{Op: "open", Path: full, Err: fs.ErrNotExist}
}

func (f *docFS) openDoc(d Doc) (fs.File, error) {
	if d.Dir {
		kids, err := f.children(d.URI)
		if err != nil {
			return nil, err
		}
		return newDir(d.Name, kids), nil
	}
	body, err := f.docs.Open(d.URI)
	if err != nil {
		return nil, err
	}
	return &docFile{info: infoOf(d), body: body}, nil
}

func (f *docFS) children(uri string) ([]Doc, error) {
	f.mu.Lock()
	if kids, ok := f.cache[uri]; ok {
		f.mu.Unlock()
		return kids, nil
	}
	f.mu.Unlock()

	kids, err := f.docs.List(uri)
	if err != nil {
		return nil, err
	}
	kids = labelDocs(kids)
	f.mu.Lock()
	defer f.mu.Unlock()
	if prev, ok := f.cache[uri]; ok {
		return prev, nil
	}
	f.cache[uri] = kids
	return kids, nil
}

func newDir(name string, kids []Doc) *docFile {
	ents := make([]fs.DirEntry, len(kids))
	for i, kid := range kids {
		ents[i] = fs.FileInfoToDirEntry(infoOf(kid))
	}
	return &docFile{info: docInfo{name: name, dir: true}, ents: ents}
}

func findDoc(docs []Doc, name string) (Doc, bool) {
	for _, d := range docs {
		if d.Name == name {
			return d, true
		}
	}
	return Doc{}, false
}

func labelDocs(docs []Doc) []Doc {
	out := make([]Doc, len(docs))
	copy(out, docs)
	seen := map[string]struct{}{}
	for i := range out {
		out[i].Name = takeName(out[i].Name, seen)
	}
	slices.SortFunc(out, func(a, b Doc) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

func takeName(name string, seen map[string]struct{}) string {
	name = cleanName(name)
	if _, ok := seen[name]; !ok {
		seen[name] = struct{}{}
		return name
	}
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for n := 2; ; n++ {
		next := stem + " (" + strconv.Itoa(n) + ")" + ext
		if _, ok := seen[next]; ok {
			continue
		}
		seen[next] = struct{}{}
		return next
	}
}

func cleanName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	if name == "" || name == "." || name == ".." {
		return "_"
	}
	return name
}

type docFile struct {
	info docInfo
	body fs.File
	ents []fs.DirEntry
	off  int
}

func (f *docFile) Stat() (fs.FileInfo, error) { return f.info, nil }

func (f *docFile) Read(p []byte) (int, error) {
	if f.info.dir {
		return 0, &fs.PathError{Op: "read", Path: f.info.name, Err: fs.ErrInvalid}
	}
	return f.body.Read(p)
}

func (f *docFile) Close() error {
	if f.body == nil {
		return nil
	}
	return f.body.Close()
}

func (f *docFile) ReadDir(n int) ([]fs.DirEntry, error) {
	if !f.info.dir {
		return nil, &fs.PathError{Op: "readdir", Path: f.info.name, Err: fs.ErrInvalid}
	}
	return pageDir(f.ents, &f.off, n)
}

type docInfo struct {
	name string
	size int64
	dir  bool
}

func infoOf(d Doc) docInfo {
	if d.Dir {
		return docInfo{name: d.Name, dir: true}
	}
	return docInfo{name: d.Name, size: d.Size}
}

func (d docInfo) Name() string { return d.name }
func (d docInfo) Size() int64 {
	if d.dir {
		return 0
	}
	return d.size
}
func (d docInfo) Mode() fs.FileMode {
	if d.dir {
		return fs.ModeDir | 0o555
	}
	return 0o444
}
func (d docInfo) ModTime() time.Time { return time.Time{} }
func (d docInfo) IsDir() bool        { return d.dir }
func (d docInfo) Sys() any           { return nil }

func pageDir(ents []fs.DirEntry, off *int, n int) ([]fs.DirEntry, error) {
	if *off >= len(ents) {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	if n <= 0 {
		out := ents[*off:]
		*off = len(ents)
		return out, nil
	}
	end := *off + n
	if end > len(ents) {
		end = len(ents)
	}
	out := ents[*off:end]
	*off = end
	return out, nil
}
