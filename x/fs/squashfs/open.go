package squashfs

import (
	"context"
	"encoding/binary"
	"errors"
	"io"
	"io/fs"

	"github.com/diskfs/go-diskfs/filesystem/squashfs"

	lewfs "github.com/lewtec/lewkit/x/fs"
)

const (
	magicLE      = 0x73717368
	superblockN  = 96
	offBlockSize = 12
	offBytesUsed = 40
)

// errFormat is a superblock that is not SquashFS.
var errFormat = errors.New("invalid squashfs")

// FS is a read-only SquashFS image.
type FS struct {
	r *squashfs.FileSystem
}

var (
	_ fs.FS         = (*FS)(nil)
	_ fs.ReadDirFS  = (*FS)(nil)
	_ fs.ReadFileFS = (*FS)(nil)
	_ fs.StatFS     = (*FS)(nil)
)

// Open reads a SquashFS image from r. r must be an [io.ReaderAt].
func Open(ctx context.Context, r io.Reader) (*FS, error) {
	if err := ctx.Err(); err != nil {
		return nil, context.Cause(ctx)
	}
	ra, err := lewfs.ReaderAt("open", r)
	if err != nil {
		return nil, err
	}
	ra = lewfs.ContextReaderAt(ctx, ra)
	var hdr [superblockN]byte
	if _, err := ra.ReadAt(hdr[:], 0); err != nil {
		return nil, err
	}
	if binary.LittleEndian.Uint32(hdr[0:4]) != magicLE {
		return nil, &fs.PathError{Op: "open", Path: "", Err: errFormat}
	}
	blocksize := int64(binary.LittleEndian.Uint32(hdr[offBlockSize : offBlockSize+4]))
	bytesUsed := int64(binary.LittleEndian.Uint64(hdr[offBytesUsed : offBytesUsed+8]))
	size, err := lewfs.Size("open", r)
	if err != nil {
		size = bytesUsed
	}
	img, err := squashfs.Read(&storage{ra: ra, n: size}, size, 0, blocksize)
	if err != nil {
		return nil, err
	}
	img.SetCacheSize(8 << 20)
	return &FS{r: img}, nil
}
