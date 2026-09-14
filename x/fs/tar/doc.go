// Package tar presents a tar archive as an [io/fs.FS].
//
// [Files] walks member headers. Tar has no central directory, so
// that walk is the listing. Each [github.com/lewtec/lewkit/x/fs.File]
// holds the name and a Reader for the body.
// [github.com/lewtec/lewkit/x/fs.File.Open] reads that body.
// [Open] makes an uncompressed [io.ReaderAt], then
// [github.com/lewtec/lewkit/x/fs.New] indexes [Files].
//
//	for f, err := range tar.Files(r) {
//		if ok, _ := f.Name.MatchGlob("**/*.go"); ok {
//			rf, err := f.Open()
//		}
//	}
//	t, err := path.OpenFS(path.New("src.tar"), root, tar.Open)
//	t, err := path.OpenFS(path.New("src.tar.gz"), root, tar.Open)
//	err = fs.Copy(ctx, t, dest, nil)
//	err = fs.Copy(ctx, t, dest, func(p path.Path) bool {
//		ok, _ := p.MatchGlob("**/*.go")
//		return ok
//	})
//
// Compressed wrappers are chosen by file name or magic through the
// process-wide [github.com/lewtec/lewkit/x/compression] registry.
// The name comes from Stat when r is an [io/fs.File]. Brotli matches
// by extension only. A compressed stream is decompressed into memory.
// An uncompressed tar still needs [io.ReaderAt].
//
// Opened files implement [io.ReaderAt] from the member offset.
// Writes are not implemented.
package tar
