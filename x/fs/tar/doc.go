// Package tar presents an uncompressed tar archive as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. The reader must also be an [io.ReaderAt].
// A missing [io.ReaderAt] returns [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
// Open does not copy or spool the archive.
//
// Tar has no central directory. Open walks member headers and seeks
// over file payloads using the size in each header.
//
//	t, err := path.OpenFS(path.New("src.tar"), root, tar.Open)
//	b, err := fs.ReadFile(t, "README")
//
// Opened files implement [io.ReaderAt] from the member offset.
// Compressed tar (.tar.gz) is not supported. Writes are not implemented.
package tar
