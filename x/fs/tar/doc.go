// Package tar presents a tar archive as an [io/fs.FS].
//
// [Files] is the listing: it walks member headers. Tar has no
// central directory, so that walk is the only index. Path scans
// ([github.com/lewtec/lewkit/x/path.Path.Select],
// [github.com/lewtec/lewkit/x/path.Path.Under]) run on
// [github.com/lewtec/lewkit/x/fs.Names] of that listing.
// [Open] indexes the listing with [github.com/lewtec/lewkit/x/fs.New].
//
// [Open] takes an [io.Reader]. An uncompressed tar must also be an
// [io.ReaderAt]. A missing [io.ReaderAt] returns
// [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
//
// Compressed wrappers are chosen by file name or magic through the
// process-wide [github.com/lewtec/lewkit/x/compression] registry.
// The name comes from Stat when r is an [io/fs.File]. Brotli matches
// by extension only. A compressed stream is decompressed into memory,
// then opened as an uncompressed tar.
//
//	for f, err := range tar.Files(r) {
//		// f.Name is a path.Path
//	}
//	t, err := path.OpenFS(path.New("src.tar"), root, tar.Open)
//	t, err := path.OpenFS(path.New("src.tar.gz"), root, tar.Open)
//	b, err := fs.ReadFile(t, "README")
//
// Opened files implement [io.ReaderAt] from the member offset.
// Writes are not implemented.
package tar
