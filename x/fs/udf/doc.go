// Package udf presents a UDF volume as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. The reader must also be an [io.ReaderAt].
// A missing [io.ReaderAt] returns [ErrNeedReadAt]. Open does not copy
// or spool the volume.
//
//	f, err := os.Open("en-us.iso")
//	vol, err := udf.Open(f)
//	b, err := fs.ReadFile(vol, "sources/install.wim")
//
// Opened files implement [io.ReaderAt] so a nested reader (WIM, zip)
// can type-assert it. Writes are not implemented.
package udf
