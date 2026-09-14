// Package tar presents a tar archive as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. An uncompressed tar must also be an
// [io.ReaderAt]. A missing [io.ReaderAt] returns
// [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
//
// Tar has no central directory. Open walks member headers and seeks
// over file payloads using the size in each header.
//
// Compressed wrappers are chosen by file name or magic through the
// process-wide [github.com/lewtec/lewkit/x/compression] registry.
// The name comes from Stat when r is an [io/fs.File]. Brotli matches
// by extension only. A compressed stream is decompressed into memory,
// then opened as an uncompressed tar.
//
//	t, err := path.OpenFS(path.New("src.tar"), root, tar.Open)
//	t, err := path.OpenFS(path.New("src.tar.gz"), root, tar.Open)
//	b, err := fs.ReadFile(t, "README")
//
// Opened files implement [io.ReaderAt] from the member offset.
// Writes are not implemented.
package tar
