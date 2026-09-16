// Package udf presents a UDF volume as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. The reader must also be an [io.ReaderAt].
// A missing [io.ReaderAt] returns [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
// Open does not copy or spool the volume.
//
//	vol, err := path.OpenFS(ctx, path.New("en-us.iso"), root, udf.Open)
//	b, err := fs.ReadFile(vol, "sources/install.wim")
//
// Opened files implement [io.ReaderAt] so a nested reader (WIM, zip,
// tar, squashfs) can type-assert it. Writes are not implemented.
package udf
