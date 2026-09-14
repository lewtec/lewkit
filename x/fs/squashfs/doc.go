// Package squashfs presents a SquashFS image as an [io/fs.FS].
//
// [Open] takes an [io.Reader]. The reader must also be an [io.ReaderAt].
// A missing [io.ReaderAt] returns [github.com/lewtec/lewkit/x/fs.ErrNeedReadAt].
// Open does not copy or spool the image.
//
// Open reads the superblock at offset 0 and seeks to the inode and
// directory tables. It does not scan the data area.
//
//	img, err := path.OpenFS(path.New("root.sfs"), root, squashfs.Open)
//	b, err := fs.ReadFile(img, "etc/os-release")
//
// Opened files are streams. They do not implement [io.ReaderAt].
// Writes are not implemented.
package squashfs
