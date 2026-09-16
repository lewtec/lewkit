// Package zip presents a ZIP archive as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. The reader must also be an [io.ReaderAt]
// with a known size. A missing [io.ReaderAt] returns
// [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt]. A missing size
// returns [github.com/lewtec/lewkit/x/fs.ErrNeedSize]. Open does not
// copy or spool the archive.
//
// Open reads the end-of-central-directory record and the central
// directory. It does not scan local file headers.
//
//	z, err := path.OpenFS(ctx, path.New("src.zip"), root, zip.Open)
//	b, err := fs.ReadFile(z, "README")
//
// Stored files implement [io.ReaderAt] from the local header offset.
// Deflated files are streams. Writes are not implemented.
package zip
